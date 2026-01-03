package processlist

import "netps/internal/process"

type initMsg struct {
	Width, Height int
}
type processSummariesLoadedMsg struct {
	ProcessSummaries []process.ProcessSummary
}

type hydrationErrorMsg struct {
	Error error
}
