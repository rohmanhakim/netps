package process

import (
	"log"
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
	aggregated := make(map[string]int)
	for _, socket := range socks {
		switch socket.State {
		case "LISTEN":
			aggregated["L"]++
		case "ESTABLISHED":
			aggregated["E"]++
		case "CLOSE":
			aggregated["C"]++
		default:
			log.Printf("Unknown socket state: %s", socket.State)
		}
	}
	p.LSocketCount = aggregated["L"]
	p.ESocketCount = aggregated["E"]
	p.CSocketCount = aggregated["C"]
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
