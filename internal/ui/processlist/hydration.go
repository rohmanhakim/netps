package processlist

import (
	"context"
	"netps/internal/process"
	"netps/internal/procfs"
	"netps/internal/sysconf"

	tea "charm.land/bubbletea/v2"
)

func HydrateRunningProcesses(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		procfsClient := procfs.NewClient()
		sysconfClient := sysconf.NewClient()

		cfg := process.Config{
			Process:   procfsClient,
			Detail:    procfsClient,
			Clocktick: sysconfClient,
			PageSize:  sysconfClient,
			UpTime:    procfsClient,
			Resource:  procfsClient,
			User:      procfsClient,
		}
		service := process.NewProcessService(cfg)
		processSummaries, err := service.GetRunningSummaries(ctx)
		if err != nil {
			return hydrationErrorMsg{Error: err}
		}
		return processSummariesLoadedMsg{
			ProcessSummaries: processSummaries,
		}
	}
}
