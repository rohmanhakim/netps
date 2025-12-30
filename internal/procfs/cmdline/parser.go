package cmdline

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseCmdLine(pid int) (string, error) {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	args := bytes.Split(data, []byte{0})
	parts := []string{}
	for _, a := range args {
		if len(a) > 0 {
			parts = append(parts, shellQuote(a))
		}
	}
	cmdline := strings.Join(parts, " ")

	return cmdline, nil
}

func shellQuote(b []byte) string {
	s := string(b)
	if strings.ContainsAny(s, " \t\n\"'\\$`") {
		return strconv.Quote(s)
	}
	return s
}
