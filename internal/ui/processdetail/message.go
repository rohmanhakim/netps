package processdetail

import (
	"netps/internal/socket"
	"netps/internal/ui/common/lifecycle"
	"time"
)

type StaticIdHydrationData struct {
	ExecPath   string
	Command    string
	PPID       int
	ParentName string
	state      lifecycle.HydrationState
	err        error
}

type ResourceHydrationData struct {
	RSSByte     int64
	StartTime   time.Duration
	ElapsedTime time.Duration
	VSZByte     uint64
	UTime       time.Duration
	STime       time.Duration
	state       lifecycle.HydrationState
	err         error
}

type UserHydrationData struct {
	UserUID        int
	UserName       string
	UserPrivileged string
	state          lifecycle.HydrationState
	err            error
}

type SocketsHydrationData struct {
	Sockets []socket.Socket
	state   lifecycle.HydrationState
	err     error
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
