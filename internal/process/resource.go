package process

import "time"

type ProcessResource struct {
	ResidentSetSizePage    int64  //pages
	VirtualMemorySize      uint64 // bytes
	StartTimeTick          uint64
	StartTimeSec           time.Duration
	ElapsedTimeSec         time.Duration
	UserCPUTimeSecond      time.Duration
	UserCPUTimeClockTick   uint64
	SystemCPUTimeSecond    time.Duration
	SystemCPUTimeClockTick uint64
}
