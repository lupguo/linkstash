package services

import (
	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
)

// VisitService provides business logic for recording visits.
type VisitService struct {
	visitRepo repos.VisitRepo
}

// NewVisitService creates a new VisitService.
func NewVisitService(visitRepo repos.VisitRepo) *VisitService {
	return &VisitService{visitRepo: visitRepo}
}

// RecordURLVisit creates a visit record for a URL.
func (s *VisitService) RecordURLVisit(urlID uint, ip, userAgent string) error {
	record := &entity.VisitRecord{
		URLID:     urlID,
		IP:        ip,
		UserAgent: userAgent,
	}
	return s.visitRepo.Create(record)
}

// RecordShortVisit creates a visit record for a short link.
func (s *VisitService) RecordShortVisit(shortID uint, ip, userAgent string) error {
	record := &entity.VisitRecord{
		ShortID:   shortID,
		IP:        ip,
		UserAgent: userAgent,
	}
	return s.visitRepo.Create(record)
}
