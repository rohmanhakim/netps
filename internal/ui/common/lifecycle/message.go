package lifecycle

// Messages that represent side effects (block these on cancellation)
type SideEffectMsg interface {
	isSideEffect()
}

// Messages that represent UI state changes (always allow these)
type UiStateMsg interface {
	isUIState()
}

type SendSignalMsg struct {
	ProcessPID  int
	ProcessName string
}

type CloseSendSignalModalMsg struct{}

func (SendSignalMsg) isUIState()           {}
func (CloseSendSignalModalMsg) isUIState() {}

type DismissnotificationMsg struct{}

type RetryMsg struct{}

func (DismissnotificationMsg) isUIState() {}
