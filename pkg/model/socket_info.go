package model

type SocketInfo struct {
	Proto string
	Addr  string
	Port  int
	State string
}

func NewSocketInfo(proto, addr string, port int, state string) *SocketInfo {
	return &SocketInfo{
		Proto: proto,
		Addr:  addr,
		Port:  port,
		State: state,
	}
}
