package application

import (
	"time"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/services"
)

// ShortURLUsecase is a thin orchestration layer that delegates to the ShortURLService.
type ShortURLUsecase struct {
	shortService *services.ShortURLService
}

// NewShortURLUsecase creates a new ShortURLUsecase with the given service.
func NewShortURLUsecase(s *services.ShortURLService) *ShortURLUsecase {
	return &ShortURLUsecase{shortService: s}
}

// GenerateShortLink delegates short link creation to the domain service.
func (uc *ShortURLUsecase) GenerateShortLink(longURL string, ttl *time.Duration) (*entity.ShortLink, error) {
	return uc.shortService.GenerateShortLink(longURL, ttl)
}

// ResolveCode delegates code resolution to the domain service.
func (uc *ShortURLUsecase) ResolveCode(code string) (*entity.ShortLink, error) {
	return uc.shortService.ResolveCode(code)
}

// ListShortLinks delegates listing to the domain service.
func (uc *ShortURLUsecase) ListShortLinks(page, size int) ([]*entity.ShortLink, int64, error) {
	return uc.shortService.ListShortLinks(page, size)
}

// DeleteShortLink delegates deletion to the domain service.
func (uc *ShortURLUsecase) DeleteShortLink(id uint) error {
	return uc.shortService.DeleteShortLink(id)
}

// RecordClick delegates click recording to the domain service.
func (uc *ShortURLUsecase) RecordClick(id uint) error {
	return uc.shortService.RecordClick(id)
}
