package process

import "context"

type ProcessSource interface {
	ListRunnings(ctx context.Context) ([]Process, error)
}

type DetailSource interface {
	Detail(ctx context.Context, pid int) (ProcessDetail, error)
}
