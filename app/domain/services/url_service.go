package services

import (
	"errors"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
)

// URLService provides business logic for URL management.
type URLService struct {
	urlRepo repos.URLRepo
}

// NewURLService creates a new URLService.
func NewURLService(urlRepo repos.URLRepo) *URLService {
	return &URLService{urlRepo: urlRepo}
}

// AddURL creates a new URL entry with status "pending".
func (s *URLService) AddURL(link string) (*entity.URL, error) {
	if link == "" {
		return nil, errors.New("link must not be empty")
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
func (s *URLService) ListURLs(page, size int, sort, category, tags string) ([]*entity.URL, int64, error) {
	return s.urlRepo.List(page, size, sort, category, tags)
}

// RecordVisit increments the visit counter for the URL with the given ID.
func (s *URLService) RecordVisit(id uint) error {
	return s.urlRepo.IncrementVisit(id)
}
