package uptime

import (
	"os"
	"strconv"
	"strings"
)

func ParseSystemUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	return strconv.ParseFloat(fields[0], 64)
}
