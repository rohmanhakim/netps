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
	user      UserSource
}

type Config struct {
	Process   SummarySource
	Detail    DetailSource
	Clocktick ClockTickSource
	PageSize  PageSizeSource
	UpTime    UpTimeSource
	Resource  ResourceSource
	User      UserSource
}

func NewProcessService(
	cfg Config,
) *Service {
	return &Service{
		process:   cfg.Process,
		detail:    cfg.Detail,
		clocktick: cfg.Clocktick,
		pageSize:  cfg.PageSize,
		upTime:    cfg.UpTime,
		resource:  cfg.Resource,
		user:      cfg.User,
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

func (s *Service) GetUser(ctx context.Context, pid int) (ProcessUser, error) {
	user, err := s.user.User(ctx, pid)
	if err != nil {
		return ProcessUser{}, nil
	}
	return user, nil
}
