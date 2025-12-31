package socket

import "context"

type Service struct {
	socket Socketsource
}

func NewService(
	socket Socketsource,
) *Service {
	return &Service{
		socket: socket,
	}
}

func (s *Service) GetSocketsByStates(ctx context.Context, pid int, states []string) ([]Socket, error) {
	sockets, err := s.socket.SocketsByStates(ctx, pid, states)
	if err != nil {
		return []Socket{}, nil
	}

	return sockets, nil
}
