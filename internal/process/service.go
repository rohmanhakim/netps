package process

import (
	"context"
	"netps/internal/procfs/uptime"
	"time"
)

type Service struct {
	process   ProcessSource
	detail    DetailSource
	clocktick ClockTickSource
	resource  ResourceSource
}

func NewService(
	proc ProcessSource,
	detail DetailSource,
	clocktick ClockTickSource,
	resource ResourceSource,
) *Service {
	return &Service{
		process:   proc,
		detail:    detail,
		clocktick: clocktick,
		resource:  resource,
	}
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

func (s *Service) GetClockTick(ctx context.Context) (int64, error) {
	clocktick, err := s.clocktick.ClockTick(ctx)
	if err != nil {
		return -1, nil
	}

	return clocktick, nil
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
	upTime, err := uptime.ParseSystemUptime()
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

	return processResource, nil
}
