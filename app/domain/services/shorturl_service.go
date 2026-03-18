package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// ShortURLService provides business logic for short link management.
type ShortURLService struct {
	shortRepo repos.ShortURLRepo
}

// NewShortURLService creates a new ShortURLService.
func NewShortURLService(shortRepo repos.ShortURLRepo) *ShortURLService {
	return &ShortURLService{shortRepo: shortRepo}
}

// GenerateShortLink creates a new short link for the given long URL.
// If ttl is provided, the link will expire after the given duration.
func (s *ShortURLService) GenerateShortLink(longURL string, ttl *time.Duration) (*entity.ShortLink, error) {
	if longURL == "" {
		return nil, fmt.Errorf("long URL must not be empty")
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		code, err := generateCode(longURL)
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}

		link := &entity.ShortLink{
			Code:    code,
			LongURL: longURL,
		}

		if ttl != nil {
			expiresAt := time.Now().Add(*ttl)
			link.ExpiresAt = &expiresAt
		}

		if err := s.shortRepo.Create(link); err != nil {
			lastErr = err
			continue // retry on collision
		}
		return link, nil
	}

	return nil, fmt.Errorf("failed to create short link after 3 attempts: %w", lastErr)
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

// ResolveCode retrieves the short link by code and checks if it has expired.
func (s *ShortURLService) ResolveCode(code string) (*entity.ShortLink, error) {
	link, err := s.shortRepo.GetByCode(code)
	if err != nil {
		return nil, fmt.Errorf("short link not found: %w", err)
	}

	if link.IsExpired() {
		return nil, fmt.Errorf("short link has expired")
	}

	return link, nil
}

// ListShortLinks returns a paginated list of short links.
func (s *ShortURLService) ListShortLinks(page, size int) ([]*entity.ShortLink, int64, error) {
	return s.shortRepo.List(page, size)
}

// DeleteShortLink soft-deletes a short link by its ID.
func (s *ShortURLService) DeleteShortLink(id uint) error {
	return s.shortRepo.Delete(id)
}

// RecordClick increments the click count for the given short link.
func (s *ShortURLService) RecordClick(id uint) error {
	return s.shortRepo.IncrementClick(id)
}
