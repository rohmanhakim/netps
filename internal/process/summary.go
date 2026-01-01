package process

import (
	"netps/internal/socket"
	"strconv"
	"strings"
)

type ProcessSummary struct {
	PID          int
	Name         string
	LSocketCount int
	ESocketCount int
	CSocketCount int
	LPortsText   string
}

func NewSummary(pid int, name string) *ProcessSummary {
	ps := ProcessSummary{
		PID:  pid,
		Name: name,
	}
	return &ps
}

func (p *ProcessSummary) WithAggregatedSockets(socks []socket.Socket) *ProcessSummary {
	aggregated := socket.Aggregate(socks)
	p.LSocketCount = aggregated.ListenCount
	p.ESocketCount = aggregated.EstablishedCount
	p.CSocketCount = aggregated.CloseCount
	return p
}

func (p *ProcessSummary) WithFilteredListenPorts(socks []socket.Socket) *ProcessSummary {
	listenPorts := []string{}
	for _, socket := range socks {
		if socket.State == "LISTEN" {
			listenPorts = append(listenPorts, strconv.Itoa(socket.Port))
		}
	}
	p.LPortsText = strings.Join(listenPorts, ",")
	return p
}
