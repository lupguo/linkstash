package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
	"gorm.io/gorm"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// URLService provides business logic for URL management.
type URLService struct {
	urlRepo repos.URLRepo
}

// NewURLService creates a new URLService.
func NewURLService(urlRepo repos.URLRepo) *URLService {
	return &URLService{urlRepo: urlRepo}
}

// AddURL creates a new URL entry with status "pending".
// If a soft-deleted record with the same link exists, it restores and resets that record instead.
func (s *URLService) AddURL(link string) (*entity.URL, error) {
	if link == "" {
		return nil, errors.New("link must not be empty")
	}

	// Check if a soft-deleted record exists for this link
	deleted, err := s.urlRepo.GetDeletedByLink(link)
	if err == nil && deleted != nil {
		// Restore the soft-deleted record and reset its status
		if err := s.urlRepo.Restore(deleted.ID); err != nil {
			return nil, fmt.Errorf("restore soft-deleted URL: %w", err)
		}
		// Reset status to pending for re-analysis
		deleted.Status = "pending"
		deleted.DeletedAt = gorm.DeletedAt{}
		if err := s.urlRepo.Update(deleted); err != nil {
			return nil, fmt.Errorf("update restored URL: %w", err)
		}
		return deleted, nil
	}

	url := &entity.URL{
		Link:   link,
		Status: "pending",
	}
	if err := s.urlRepo.Create(url); err != nil {
		return nil, err
	}
	return url, nil
}

// GetURL retrieves a URL by its ID.
func (s *URLService) GetURL(id uint) (*entity.URL, error) {
	return s.urlRepo.GetByID(id)
}

// UpdateURL updates an existing URL record.
func (s *URLService) UpdateURL(url *entity.URL) error {
	return s.urlRepo.Update(url)
}

// DeleteURL soft-deletes a URL by its ID.
func (s *URLService) DeleteURL(id uint) error {
	return s.urlRepo.Delete(id)
}

// ListURLs returns a paginated, sorted, and filtered list of URLs and total count.
func (s *URLService) ListURLs(page, size int, sort, category, tags string, isShortURL bool, networkType string) ([]*entity.URL, int64, error) {
	return s.urlRepo.List(page, size, sort, category, tags, isShortURL, networkType)
}

// RecordVisit increments the visit counter for the URL with the given ID.
func (s *URLService) RecordVisit(id uint) error {
	return s.urlRepo.IncrementVisit(id)
}

// --- Short link methods ---

// GenerateShortLink finds or creates a URL record for the given longURL, then assigns a short code.
// If customCode is provided, it is used (must be unique); otherwise a random code is generated.
// If ttl is provided, the short code will expire after the given duration.
func (s *URLService) GenerateShortLink(longURL, customCode string, ttl *time.Duration) (*entity.URL, error) {
	if longURL == "" {
		return nil, fmt.Errorf("long URL must not be empty")
	}

	// Find or create the URL record
	url, err := s.urlRepo.GetByLink(longURL)
	if err != nil {
		// Not found — check for soft-deleted record first
		deleted, delErr := s.urlRepo.GetDeletedByLink(longURL)
		if delErr == nil && deleted != nil {
			if err := s.urlRepo.Restore(deleted.ID); err != nil {
				return nil, fmt.Errorf("failed to restore URL record: %w", err)
			}
			deleted.Status = "pending"
			deleted.DeletedAt = gorm.DeletedAt{}
			url = deleted
		} else {
			// Create a new URL record
			url = &entity.URL{
				Link:   longURL,
				Status: "pending",
			}
			if err := s.urlRepo.Create(url); err != nil {
				return nil, fmt.Errorf("failed to create URL record: %w", err)
			}
		}
	}

	// If the URL already has a short code, return it directly
	if url.HasShortCode() {
		return url, nil
	}

	// Determine short code
	if customCode != "" {
		// Check uniqueness
		existing, err := s.urlRepo.GetByShortCode(customCode)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("code '%s' is already in use", customCode)
		}
		url.ShortCode = customCode
	} else {
		// Auto-generate with retry
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			code, err := generateCode(longURL)
			if err != nil {
				return nil, fmt.Errorf("failed to generate code: %w", err)
			}
			// Check uniqueness
			existing, err := s.urlRepo.GetByShortCode(code)
			if err == nil && existing != nil {
				lastErr = fmt.Errorf("code '%s' collision", code)
				continue
			}
			url.ShortCode = code
			lastErr = nil
			break
		}
		if lastErr != nil {
			return nil, fmt.Errorf("failed to generate unique code after 3 attempts: %w", lastErr)
		}
	}

	// Set expiration
	if ttl != nil {
		expiresAt := time.Now().Add(*ttl)
		url.ShortExpiresAt = &expiresAt
	}

	if err := s.urlRepo.Update(url); err != nil {
		return nil, fmt.Errorf("failed to save short code: %w", err)
	}
	return url, nil
}

// ResolveShortCode retrieves the URL by short code and checks expiration.
func (s *URLService) ResolveShortCode(code string) (*entity.URL, error) {
	url, err := s.urlRepo.GetByShortCode(code)
	if err != nil {
		return nil, fmt.Errorf("short link not found: %w", err)
	}

	if url.IsShortExpired() {
		return nil, fmt.Errorf("short link has expired")
	}

	return url, nil
}

// UpdateShortLink updates the short code and/or long URL of a URL record.
// It checks code uniqueness when the code is being changed.
func (s *URLService) UpdateShortLink(id uint, newCode, newLongURL string) (*entity.URL, error) {
	url, err := s.urlRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("URL not found: %w", err)
	}

	// If code is changing, check uniqueness
	if newCode != "" && newCode != url.ShortCode {
		existing, err := s.urlRepo.GetByShortCode(newCode)
		if err == nil && existing.ID != id {
			return nil, fmt.Errorf("code '%s' is already in use", newCode)
		}
		url.ShortCode = newCode
	}

	if newLongURL != "" {
		url.Link = newLongURL
	}

	if err := s.urlRepo.Update(url); err != nil {
		return nil, fmt.Errorf("failed to update URL: %w", err)
	}
	return url, nil
}

// ListShortLinks returns a paginated list of URLs that have short codes.
func (s *URLService) ListShortLinks(page, size int) ([]*entity.URL, int64, error) {
	return s.urlRepo.ListByShortCode(page, size)
}

// ClearShortCode removes the short code from a URL without deleting the URL record.
func (s *URLService) ClearShortCode(id uint) error {
	return s.urlRepo.ClearShortCode(id)
}

// generateCode generates a 6-character Base62 code from the long URL.
func generateCode(longURL string) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	data := fmt.Sprintf("%s%d%x", longURL, time.Now().UnixNano(), randomBytes)
	hash := sha256.Sum256([]byte(data))

	// Take first 6 bytes, convert to uint64, mod 62^6
	num := binary.BigEndian.Uint64(append([]byte{0, 0}, hash[:6]...))
	const mod = 62 * 62 * 62 * 62 * 62 * 62 // 62^6

	num = num % mod

	code := make([]byte, 6)
	for i := 5; i >= 0; i-- {
		code[i] = base62Chars[num%62]
		num /= 62
	}

	return string(code), nil
}
