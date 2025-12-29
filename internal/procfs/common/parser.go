package common

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func ParseSingleLineProcFd(pid int, fdname string) (string, error) {
	var sb strings.Builder

	_, err := sb.WriteString("/proc/")
	if err != nil {
		return "", err
	}
	_, err = sb.WriteString(strconv.Itoa(pid))
	if err != nil {
		return "", err
	}
	_, err = sb.WriteString("/" + fdname)
	if err != nil {
		return "", err
	}

	f, err := os.Open(sb.String())
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return "", nil
	}
	var name string

	name = scanner.Text()

	return name, nil
}
