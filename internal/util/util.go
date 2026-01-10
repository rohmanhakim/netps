package util

import (
	"bufio"
	"fmt"
	"os"
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

func ReadFirstLine(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	// Ensure the file is closed after the function finishes
	defer file.Close()

	// Create a new scanner for the file
	scanner := bufio.NewScanner(file)

	// Scan the first line
	if scanner.Scan() {
		return scanner.Text(), nil
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error during scan: %w", err)
	}

	// If no line was found (e.g., the file is empty)
	return "", fmt.Errorf("file is empty, no lines found")
}
