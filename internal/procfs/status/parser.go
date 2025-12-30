package status

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// RealUID returns the real (owner) UID of the process.
func ParseRealUID(pid int) (int, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Format:
		// Uid:    1000    1000    1000    1000
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return 0, fmt.Errorf("malformed Uid line: %q", line)
			}
			return strconv.Atoi(fields[1]) // Real UID
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return 0, fmt.Errorf("Uid field not found")
}

func ParseEffectiveUID(pid int) (int, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return 0, fmt.Errorf("malformed Uid line")
			}
			return strconv.Atoi(fields[2]) // Effective UID
		}
	}
	return 0, fmt.Errorf("Uid field not found")
}
