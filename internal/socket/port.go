package socket

import "context"

type Socketsource interface {
	SocketsByStates(ctx context.Context, pid int, states []SocketState) ([]Socket, error)
}
