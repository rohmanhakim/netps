package stat

type Stat struct {
	PID       int
	Comm      string //
	State     string // process state name
	PPID      int    // parent PID
	UTime     uint64 // CPU time in user mode (clock ticks)
	STime     uint64 // CPU time in kernel mode
	StartTime uint64 // process start time since boot (ticks)
	VSize     uint64 // virtual memory size (bytes)
	RSS       int64  // resident set size (pages)
	Processor int    // last CPU the process ran on
}
