package process

import (
	"context"
	"time"
)

type Service struct {
	process   SummarySource
	detail    DetailSource
	clocktick ClockTickSource
	pageSize  PageSizeSource
	upTime    UpTimeSource
	resource  ResourceSource
}

func NewService(
	proc SummarySource,
	detail DetailSource,
	clocktick ClockTickSource,
	pageSize PageSizeSource,
	upTime UpTimeSource,
	resource ResourceSource,
) *Service {
	return &Service{
		process:   proc,
		detail:    detail,
		clocktick: clocktick,
		pageSize:  pageSize,
		upTime:    upTime,
		resource:  resource,
	}
}

func (s *Service) GetRunningSummaries(ctx context.Context) ([]ProcessSummary, error) {
	procSum, err := s.process.ListRunnings(ctx)

	if err != nil {
		return nil, err
	}

	out := make([]ProcessSummary, 0, len(procSum))
	for _, pr := range procSum {
		out = append(out, pr)
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

func (s *Service) GetClockTick(ctx context.Context) (int64, error) {
	clocktick, err := s.clocktick.ClockTick(ctx)
	if err != nil {
		return -1, nil
	}

	return clocktick, nil
}

func (s *Service) GetPageSize(ctx context.Context) (int64, error) {
	pageSize, err := s.pageSize.PageSize(ctx)
	if err != nil {
		return -1, nil
	}

	return pageSize, nil
}

func (s *Service) GetProcessResource(ctx context.Context, pid int) (ProcessResource, error) {
	processResource, err := s.resource.Resource(ctx, pid)
	if err != nil {
		return ProcessResource{}, err
	}
	sysClockTick, err := s.clocktick.ClockTick(ctx)
	if err != nil {
		return ProcessResource{}, err
	}

	startTimeSec := (processResource.StartTimeTick) / uint64(sysClockTick)
	upTime, err := s.upTime.UpTime(ctx)
	if err != nil {
		return ProcessResource{}, nil
	}
	elapsedTime := upTime - float64(startTimeSec)
	uCpuTimeSec := (processResource.UserCPUTimeClockTick / uint64(sysClockTick))
	sCpuTimeSec := (processResource.SystemCPUTimeClockTick / uint64(sysClockTick))

	processResource.StartTimeSec = time.Duration(startTimeSec) * time.Second
	processResource.ElapsedTimeSec = time.Duration(elapsedTime) * time.Second
	processResource.UserCPUTimeSecond = time.Duration(uCpuTimeSec) * time.Second
	processResource.SystemCPUTimeSecond = time.Duration(sCpuTimeSec) * time.Second

	pageSize, err := s.pageSize.PageSize(ctx)
	if err != nil {
		return ProcessResource{}, err
	}

	processResource.ResidentSetSizeByte = processResource.ResidentSetSizePage * pageSize
	return processResource, nil
}
