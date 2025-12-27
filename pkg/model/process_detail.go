package model

import (
	"log"
	"strconv"
	"time"
)

type ProcessDetail struct {
	Name     string
	PPID     int
	ExecPath string
	Command  string

	Sockets []SocketInfo

	User       string
	UID        int
	GID        int
	Privileged bool

	CPUPercent float64
	MemRSSKB   int64
	Threads    int
	StartTime  time.Time
}

func NewProcessDetail() *ProcessDetail {
	return &ProcessDetail{
		ExecPath: "",
		PPID:     0,
		Command:  "",

		Sockets: []SocketInfo{},

		User:       "",
		UID:        0,
		GID:        0,
		Privileged: false,

		CPUPercent: 0.0,
		MemRSSKB:   0,
		Threads:    0,
		StartTime:  time.Time{},
	}
}

func (pd *ProcessDetail) AggregateSockets() map[string]int {
	aggregated := make(map[string]int)
	for _, socket := range pd.Sockets {
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

func (pd *ProcessDetail) FilterListenPorts() []string {
	listenPorts := []string{}
	for _, socket := range pd.Sockets {
		if socket.State == "LISTEN" {
			listenPorts = append(listenPorts, strconv.Itoa(socket.Port))
		}
	}
	return listenPorts
}
