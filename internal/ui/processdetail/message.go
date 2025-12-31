package processdetail

import (
	"netps/internal/socket"
	"time"
)

type detailHydratedMsg struct {
	ExecPath   string
	Command    string
	PPID       int
	ParentName string
	Err        error
}

type resourceHydratedMsg struct {
	RSSByte     int64
	StartTime   time.Duration
	ElapsedTime time.Duration
	VSZByte     uint64
	UTime       time.Duration
	STime       time.Duration
}

type userHydratedMsg struct {
	UserUID        int
	UserName       string
	UserPrivileged string
}

type socketHydrateMsg struct {
	Sockets []socket.Socket
}
