package processlist

import (
	"context"
	"netps/internal/process"

	tea "charm.land/bubbletea/v2"
)

func HydrateRunningProcesses(ctx context.Context, processService *process.Service) tea.Cmd {
	return func() tea.Msg {
		if ctx.Err() != nil {
			return processSummariesHydratedMsg{err: ctx.Err()} // Propagate error
		}

		processSummaries, err := processService.GetRunningSummaries(ctx)

		msg := processSummariesHydratedMsg{}
		if err == nil {
			msg = processSummariesHydratedMsg{
				processSummaries: processSummaries,
			}
		} else {
			msg.err = err
		}
		return msg
	}
}
