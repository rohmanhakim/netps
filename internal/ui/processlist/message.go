package processlist

import (
	"netps/internal/process"
)

type initMsg struct {
	Width, Height int
}

type processSummariesHydratedMsg struct {
	processSummaries []process.ProcessSummary

	err error
}
