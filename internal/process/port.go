package process

import "context"

type SummarySource interface {
	ListRunnings(ctx context.Context) ([]ProcessSummary, error)
}

type DetailSource interface {
	Detail(ctx context.Context, pid int) (ProcessDetail, error)
}

type ClockTickSource interface {
	ClockTick(ctx context.Context) (int64, error)
}

type ResourceSource interface {
	Resource(ctx context.Context, pid int) (ProcessResource, error)
}

type UpTimeSource interface {
	UpTime(ctx context.Context) (float64, error)
}

type PageSizeSource interface {
	PageSize(ctx context.Context) (int64, error)
}
