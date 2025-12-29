package comm

import "netps/internal/procfs/common"

func ParseProcessName(pid int) (string, error) {
	return common.ParseSingleLineProcFd(pid, "comm")
}
