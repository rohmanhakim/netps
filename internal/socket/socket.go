package socket

type Socket struct {
	Proto string
	Addr  string
	Port  int
	State string
}

func NewSocketInfo(proto, addr string, port int, state string) *Socket {
	return &Socket{
		Proto: proto,
		Addr:  addr,
		Port:  port,
		State: state,
	}
}
