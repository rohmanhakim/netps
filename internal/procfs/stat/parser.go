package stat

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseStat(pid int) (*Stat, error) {
	path := fmt.Sprintf("/proc/%d/stat", pid)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	line := string(data)

	// 1. Find "(comm)" safely
	open := strings.IndexByte(line, '(')
	close := strings.LastIndexByte(line, ')')
	if open < 0 || close < 0 || close <= open {
		return nil, errors.New("invalid stat format: comm")
	}

	// 2. Parse fixed parts
	pidStr := strings.TrimSpace(line[:open])
	comm := line[open+1 : close]

	pidParsed, err := strconv.Atoi(pidStr)
	if err != nil {
		return nil, err
	}

	// 3. Remaining fields (after ") ")
	fields := strings.Fields(line[close+1:])
	if len(fields) < 40 {
		return nil, errors.New("invalid stat format: too few fields")
	}

	// Field numbers are from `man proc` (1-based, comm is #2)
	stateByte := fields[0][0]
	stateName := mapProcessState(string(stateByte))
	ppid := mustInt(fields[1])
	utime := mustUint(fields[11])
	stime := mustUint(fields[12])
	startTime := mustUint(fields[19])
	vsize := mustUint(fields[20])
	rss := mustInt64(fields[21])
	processor := mustInt(fields[36])

	return &Stat{
		PID:       pidParsed,
		Comm:      comm,
		State:     stateName,
		PPID:      ppid,
		UTime:     utime,
		STime:     stime,
		StartTime: startTime,
		VSize:     vsize,
		RSS:       rss,
		Processor: processor,
	}, nil
}

func mustInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func mustInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func mustUint(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

func mapProcessState(stateChar string) string {
	switch stateChar {
	case "R":
		return "Running"
	case "S":
		return "Sleeping"
	case "D":
		return "Disk Sleep"
	case "Z":
		return "Zombie"
	case "T":
		return "Stopped"
	case "t":
		return "Tracing stop"
	case "X":
		return "Dead"
	case "I":
		return "Idle"
	default:
		return "UNKNOWN"
	}
}
