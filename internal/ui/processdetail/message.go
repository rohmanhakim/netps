package processdetail

import (
	"netps/internal/socket"
	"time"
)

type HydrationState int

const (
	StateNotAsked HydrationState = iota
	StateHydrating
	StateSuccess
	StateError
)

// Messages that represent side effects (block these on cancellation)
type sideEffectMsg interface {
	isSideEffect()
}

// Messages that represent UI state changes (always allow these)
type uiStateMsg interface {
	isUIState()
}

type StaticIdHydrationData struct {
	ExecPath   string
	Command    string
	PPID       int
	ParentName string
	state      HydrationState
	err        error
}

type ResourceHydrationData struct {
	RSSByte     int64
	StartTime   time.Duration
	ElapsedTime time.Duration
	VSZByte     uint64
	UTime       time.Duration
	STime       time.Duration
	state       HydrationState
	err         error
}

type UserHydrationData struct {
	UserUID        int
	UserName       string
	UserPrivileged string
	state          HydrationState
	err            error
}

type SocketsHydrationData struct {
	Sockets []socket.Socket
	state   HydrationState
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

type sendSignalMsg struct{}

type closeSendSignalModalMsg struct{}

type dismissnotificationMsg struct{}

type retryMsg struct{}

func (initMsg) isSideEffect()             {}
func (staticIdHydratedMsg) isSideEffect() {}
func (resourceHydratedMsg) isSideEffect() {}
func (userHydratedMsg) isSideEffect()     {}
func (socketsHydratedMsg) isSideEffect()  {}

func (sendSignalMsg) isUIState()           {}
func (closeSendSignalModalMsg) isUIState() {}
func (dismissnotificationMsg) isUIState()  {}
