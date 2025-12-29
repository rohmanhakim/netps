package process

type ProcessSummary struct {
	PID          int
	Name         string
	LSocketCount int
	ESocketCount int
	CSocketCount int
	LPortsText   string
}
