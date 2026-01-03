package processdetail

import (
	"context"
	"netps/internal/process"
	"netps/internal/procfs"
	"netps/internal/socket"
	"netps/internal/sysconf"

	tea "charm.land/bubbletea/v2"
)

func HydrateStaticIds(pid int) tea.Cmd {
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
		processService := process.NewProcessService(cfg)
		processDetail, err := processService.GetProcessDetail(context.Background(), pid)
		if err != nil {
			panic(err)
		}
		return detailHydratedMsg{
			ExecPath:   processDetail.ExecPath,
			Command:    processDetail.Command,
			PPID:       processDetail.PPID,
			ParentName: processDetail.ParentName,

			Err: err,
		}
	}
}

func HydrateResource(pid int) tea.Cmd {
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
		processService := process.NewProcessService(cfg)
		processResource, err := processService.GetProcessResource(context.Background(), pid)
		if err != nil {
			panic(err)
		}
		return resourceHydratedMsg{
			RSSByte:     processResource.ResidentSetSizeByte,
			StartTime:   processResource.StartTimeSec,
			ElapsedTime: processResource.ElapsedTimeSec,
			VSZByte:     processResource.VirtualMemorySize,
			UTime:       processResource.UserCPUTimeSecond,
			STime:       processResource.SystemCPUTimeSecond,
		}
	}
}

func HydrateUser(pid int) tea.Cmd {
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
		processService := process.NewProcessService(cfg)

		processUser, err := processService.GetUser(context.Background(), pid)
		if err != nil {
			panic(err)
		}
		return userHydratedMsg{
			UserUID:        processUser.RealUID,
			UserName:       processUser.Name,
			UserPrivileged: processUser.PrivilegedString(),
		}
	}
}

func HydrateSockets(pid int) tea.Cmd {
	return func() tea.Msg {
		procfsClient := procfs.NewClient()
		socketService := socket.NewService(procfsClient)
		socketStates := []string{"LISTEN", "ESTABLISHED", "CLOSE"}
		sockets, err := socketService.GetSocketsByStates(context.Background(), pid, socketStates)
		if err != nil {
			panic(err)
		}
		return socketHydrateMsg{
			Sockets: sockets,
		}
	}
}
