package application

import (
	"time"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/services"
)

// URLUsecase is a thin orchestration layer that delegates to the domain service.
type URLUsecase struct {
	urlService *services.URLService
}

// NewURLUsecase creates a new URLUsecase with the given URL service.
func NewURLUsecase(s *services.URLService) *URLUsecase {
	return &URLUsecase{urlService: s}
}

// AddURL delegates URL creation to the domain service.
func (uc *URLUsecase) AddURL(link string) (*entity.URL, error) {
	return uc.urlService.AddURL(link)
}

// GetURL delegates URL retrieval to the domain service.
func (uc *URLUsecase) GetURL(id uint) (*entity.URL, error) {
	return uc.urlService.GetURL(id)
}

// UpdateURL delegates URL update to the domain service.
func (uc *URLUsecase) UpdateURL(url *entity.URL) error {
	return uc.urlService.UpdateURL(url)
}

// DeleteURL soft-deletes a URL by its ID.
func (uc *URLUsecase) DeleteURL(id uint) error {
	return uc.urlService.DeleteURL(id)
}

// ListURLs returns a paginated, sorted, and filtered list of URLs.
func (uc *URLUsecase) ListURLs(page, size int, sort, category, tags string, isShortURL bool, networkType string) ([]*entity.URL, int64, error) {
	return uc.urlService.ListURLs(page, size, sort, category, tags, isShortURL, networkType)
}

// RecordVisit increments the visit counter for the URL.
func (uc *URLUsecase) RecordVisit(id uint) error {
	return uc.urlService.RecordVisit(id)
}

// --- Short link pass-through methods ---

// GenerateShortLink delegates short link creation to the domain service.
func (uc *URLUsecase) GenerateShortLink(longURL, customCode string, ttl *time.Duration) (*entity.URL, error) {
	return uc.urlService.GenerateShortLink(longURL, customCode, ttl)
}

// ResolveShortCode delegates code resolution to the domain service.
func (uc *URLUsecase) ResolveShortCode(code string) (*entity.URL, error) {
	return uc.urlService.ResolveShortCode(code)
}

// UpdateShortLink delegates updating code/long_url to the domain service.
func (uc *URLUsecase) UpdateShortLink(id uint, code, longURL string) (*entity.URL, error) {
	return uc.urlService.UpdateShortLink(id, code, longURL)
}

// ListShortLinks delegates listing short links to the domain service.
func (uc *URLUsecase) ListShortLinks(page, size int) ([]*entity.URL, int64, error) {
	return uc.urlService.ListShortLinks(page, size)
}

// ClearShortCode removes the short code from a URL without deleting the URL record.
func (uc *URLUsecase) ClearShortCode(id uint) error {
	return uc.urlService.ClearShortCode(id)
}
