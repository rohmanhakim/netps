package process

import (
	"context"
)

type Service struct {
	process ProcessSource
	detail  DetailSource
}

func NewService(proc ProcessSource, detail DetailSource) *Service {
	return &Service{process: proc, detail: detail}
}

func (s *Service) GetRunningSummaries(ctx context.Context) ([]ProcessSummary, error) {
	processes, err := s.process.ListRunnings(ctx)

	if err != nil {
		return nil, err
	}

	out := make([]ProcessSummary, 0, len(processes))
	for _, pr := range processes {
		out = append(out, pr.ToSumary())
	}

	return out, nil
}

func (s *Service) GetProcessDetail(ctx context.Context, pid int) (ProcessDetail, error) {
	processDetail, err := s.detail.Detail(ctx, pid)
	if err != nil {
		return ProcessDetail{}, nil
	}

	return processDetail, nil
}
