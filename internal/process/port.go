package process

import "context"

type ProcessSource interface {
	ListRunnings(ctx context.Context) ([]Process, error)
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

type PageSizeSource interface {
	PageSize(ctx context.Context) (int64, error)
}
