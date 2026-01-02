package processlist

import "netps/internal/process"

type processSummariesLoadedMsg struct {
	ProcessSummaries []process.ProcessSummary
}

type hydrationErrorMsg struct {
	Error error
}
