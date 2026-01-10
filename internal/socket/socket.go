package socket

type SocketState string

const (
	StateEstablished SocketState = "ESTABLISHED"
	StateSynSent     SocketState = "SYN_SENT"
	StateSynRecv     SocketState = "SYN_RECV"
	StateFinWait1    SocketState = "FIN_WAIT1"
	StateFinWait2    SocketState = "FIN_WAIT2"
	StateTimeWait1   SocketState = "TIME_WAIT"
	StateClose       SocketState = "CLOSE"
	StateCloseWait   SocketState = "CLOSE_WAIT"
	StateLastAck     SocketState = "LAST_ACK"
	StateListen      SocketState = "LISTEN"
	StateClosing     SocketState = "CLOSING"
	StateNewSynRecv  SocketState = "NEW_SYN_RECV"
	StateUnknown     SocketState = "UNKNOWN"
)

type Socket struct {
	Proto string
	Addr  string
	Port  int
	State SocketState
}

type AggregatedSockets struct {
	EstablishedCount int
	ListenCount      int
	CloseCount       int
}

func NewSocketInfo(proto, addr string, port int, state SocketState) *Socket {
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
		case StateListen:
			aggregated.ListenCount++
		case StateEstablished:
			aggregated.EstablishedCount++
		case StateClose:
			aggregated.CloseCount++
		}
	}
	return aggregated
}
