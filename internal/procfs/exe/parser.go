package exe

import (
	"fmt"
	"os"
)

func ParseProcExe(pid int) (string, error) {
	exePath, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		panic(err)
	}

	return exePath, nil
}
