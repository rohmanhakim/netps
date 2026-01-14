package processlist

import (
	"netps/internal/process"
	"netps/internal/ui/common/lifecycle"
)

type initMsg struct {
	Width, Height int
}

type processSummariesHydratedMsg struct {
	processSummaries []process.ProcessSummary

	err error
}

type processSummariesHydrationData struct {
	processSummaries []process.ProcessSummary
	state            lifecycle.HydrationState
	err              error
}
