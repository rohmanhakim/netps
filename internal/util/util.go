package util

import (
	"fmt"
	"time"
)

func DurationToHHMMSS(d time.Duration) string {
	d = d.Round(time.Second)

	// // Extract components
	hour := int(d.Hours())
	minute := int(d.Minutes()) % 60 // Use modulo 60 to keep minutes within 0-59 range
	second := int(d.Seconds()) % 60 // Use modulo 60 to keep seconds within 0-59 range

	// Format with leading zeros (e.g., 01:05:09)
	return fmt.Sprintf("%02d:%02d:%02d", hour, minute, second)
}
