package application

import (
	"context"

	"github.com/lupguo/linkstash/app/domain/services"
)

type AnalysisUsecase struct {
	worker *services.WorkerService
}

func NewAnalysisUsecase(w *services.WorkerService) *AnalysisUsecase {
	return &AnalysisUsecase{worker: w}
}

func (uc *AnalysisUsecase) EnqueueAnalysis(urlID uint) {
	uc.worker.Enqueue(urlID)
}

func (uc *AnalysisUsecase) Start(ctx context.Context) {
	uc.worker.RecoverPending()
	uc.worker.Start(ctx)
}
