package model

import (
	"time"
)

type Process struct {
	PID    int
	Detail ProcessDetail
}

func NewProcess(
	pid int,
	name string,
) *Process {
	p := &Process{
		PID: pid,
		Detail: ProcessDetail{
			Name:      name,
			PPID:      0,
			Command:   "",
			Sockets:   []SocketInfo{},
			User:      "",
			UID:       0,
			GID:       0,
			StartTime: time.Time{},
		},
	}
	return p
}
