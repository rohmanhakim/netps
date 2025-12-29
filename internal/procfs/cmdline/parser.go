package cmdline

import "netps/internal/procfs/common"

func ParseCmdLine(pid int) (string, error) {
	return common.ParseSingleLineProcFd(pid, "cmdline")
}
