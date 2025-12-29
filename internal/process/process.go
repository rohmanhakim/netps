package process

import (
	"log"
	"netps/internal/socket"
	"strconv"
	"strings"
	"time"
)

type Process struct {
	PID    int
	Name   string
	PPID   int
	Detail ProcessDetail

	User       string
	UID        int
	GID        int
	Privileged bool

	CPUPercent float64
	MemRSSKB   int64
	Threads    int
	StartTime  time.Time

	Sockets []socket.Socket
}

func NewProcess(
	pid int,
	name string,
) *Process {
	p := &Process{
		PID:  pid,
		Name: name,
		PPID: 0,
		Detail: ProcessDetail{
			ExecPath: "",
			Command:  "",
		},

		User:       "",
		UID:        0,
		GID:        0,
		Privileged: false,

		CPUPercent: 0,
		MemRSSKB:   0,
		Threads:    0,
		StartTime:  time.Time{},

		Sockets: []socket.Socket{},
	}
	return p
}

func (p *Process) AggregateSockets() map[string]int {
	aggregated := make(map[string]int)
	for _, socket := range p.Sockets {
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
	return aggregated
}

func (p *Process) FilterListenPorts() []int {
	listenPorts := []int{}
	for _, socket := range p.Sockets {
		if socket.State == "LISTEN" {
			listenPorts = append(listenPorts, socket.Port)
		}
	}
	return listenPorts
}

func (p *Process) WithSockets(sockets []socket.Socket) *Process {
	p.Sockets = sockets
	return p
}

func (p *Process) ToSumary() ProcessSummary {
	aggregatedSockets := p.AggregateSockets()

	return ProcessSummary{
		PID:          p.PID,
		Name:         p.Name,
		LSocketCount: aggregatedSockets["L"],
		ESocketCount: aggregatedSockets["E"],
		CSocketCount: aggregatedSockets["C"],
		LPortsText:   p.JoinedLPorts(),
	}
}

func (p *Process) JoinedLPorts() string {

	listenPorts := []string{}
	for _, socket := range p.Sockets {
		if socket.State == "LISTEN" {
			listenPorts = append(listenPorts, strconv.Itoa(socket.Port))
		}
	}
	return strings.Join(listenPorts, ",")
}

func (p *Process) WithDetail(execPath string, cmdText string) *Process {
	p.Detail.ExecPath = execPath
	p.Detail.Command = cmdText
	return p
}
