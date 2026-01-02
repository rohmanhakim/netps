package socket

type Socket struct {
	Proto string
	Addr  string
	Port  int
	State string
}

type AggregatedSockets struct {
	EstablishedCount int
	ListenCount      int
	CloseCount       int
}

func NewSocketInfo(proto, addr string, port int, state string) *Socket {
	return &Socket{
		Proto: proto,
		Addr:  addr,
		Port:  port,
		State: state,
	}
}

func Aggregate(socks []Socket) AggregatedSockets {
	aggregated := AggregatedSockets{}
	for _, socket := range socks {
		switch socket.State {
		case "LISTEN":
			aggregated.ListenCount++
		case "ESTABLISHED":
			aggregated.EstablishedCount++
		case "CLOSE":
			aggregated.CloseCount++
		}
	}
	return aggregated
}
