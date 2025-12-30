package comm

import (
	"bufio"
	"fmt"
	"os"
)

func ParseProcessName(pid int) (string, error) {
	path := fmt.Sprintf("/proc/%d/comm", pid)

	f, err := os.Open(path)
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
