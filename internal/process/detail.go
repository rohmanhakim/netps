package process

type ProcessDetail struct {
	ExecPath   string
	Command    string
	PPID       int
	ParentName string
}
