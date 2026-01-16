package processdetail

import (
	"netps/internal/socket"
	"time"
)

type StaticIdData struct {
	ExecPath   string
	Command    string
	PPID       int
	ParentName string
}

type ResourceData struct {
	RSSByte     int64
	StartTime   time.Duration
	ElapsedTime time.Duration
	VSZByte     uint64
	UTime       time.Duration
	STime       time.Duration
}

type UserData struct {
	UserUID        int
	UserName       string
	UserPrivileged string
}

type SocketsData struct {
	Sockets []socket.Socket
}

type staticIdHydratedMsg struct {
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
	Err         error
}

type userHydratedMsg struct {
	UserUID        int
	UserName       string
	UserPrivileged string
	Err            error
}

type socketsHydratedMsg struct {
	Sockets []socket.Socket
	Err     error
}

type initMsg struct {
	pid           int
	name          string
	content       string
	width, height int
}

func (initMsg) isSideEffect()             {}
func (staticIdHydratedMsg) isSideEffect() {}
func (resourceHydratedMsg) isSideEffect() {}
func (userHydratedMsg) isSideEffect()     {}
func (socketsHydratedMsg) isSideEffect()  {}
