package processdetail

import (
	"context"
	"netps/internal/process"
	"netps/internal/socket"

	tea "charm.land/bubbletea/v2"
)

func HydrateStaticIds(ctx context.Context, pid int, processService *process.Service) tea.Cmd {
	return func() tea.Msg {
		if ctx.Err() != nil {
			return staticIdHydratedMsg{Err: ctx.Err()} // Propagate error
		}

		processDetail, err := processService.GetProcessDetail(ctx, pid)

		msg := staticIdHydratedMsg{}
		if err == nil {
			msg = staticIdHydratedMsg{
				ExecPath:   processDetail.ExecPath,
				Command:    processDetail.Command,
				PPID:       processDetail.PPID,
				ParentName: processDetail.ParentName,
			}
		} else {
			msg.Err = err
		}
		return msg
	}
}

func HydrateResource(ctx context.Context, pid int, processService *process.Service) tea.Cmd {
	return func() tea.Msg {
		if ctx.Err() != nil {
			return resourceHydratedMsg{Err: ctx.Err()} // Propagate error
		}

		processResource, err := processService.GetProcessResource(ctx, pid)

		msg := resourceHydratedMsg{}
		if err == nil {
			msg = resourceHydratedMsg{
				RSSByte:     processResource.ResidentSetSizeByte,
				StartTime:   processResource.StartTimeSec,
				ElapsedTime: processResource.ElapsedTimeSec,
				VSZByte:     processResource.VirtualMemorySize,
				UTime:       processResource.UserCPUTimeSecond,
				STime:       processResource.SystemCPUTimeSecond,
			}
		} else {
			msg.Err = err
		}
		return msg
	}
}

func HydrateUser(ctx context.Context, pid int, processService *process.Service) tea.Cmd {
	return func() tea.Msg {
		if ctx.Err() != nil {
			return userHydratedMsg{Err: ctx.Err()} // Propagate error
		}

		processUser, err := processService.GetUser(ctx, pid)

		msg := userHydratedMsg{}
		if err == nil {
			msg = userHydratedMsg{
				UserUID:        processUser.RealUID,
				UserName:       processUser.Name,
				UserPrivileged: processUser.PrivilegedString(),
				Err:            err,
			}
		} else {
			msg.Err = err
		}
		return msg
	}
}

func HydrateSockets(ctx context.Context, pid int, socketService *socket.Service) tea.Cmd {
	return func() tea.Msg {
		if ctx.Err() != nil {
			return socketsHydratedMsg{Err: ctx.Err()} // Propagate error
		}

		socketStates := []socket.SocketState{socket.StateListen, socket.StateEstablished, socket.StateClose}
		sockets, err := socketService.GetSocketsByStates(ctx, pid, socketStates)

		msg := socketsHydratedMsg{}

		if err == nil {
			msg = socketsHydratedMsg{
				Sockets: sockets,
				Err:     err,
			}
		} else {
			msg.Err = err
		}
		return msg
	}
}

func Initialize(pid int, name string, w, h int) tea.Cmd {
	return func() tea.Msg {
		return initMsg{
			pid:    pid,
			name:   name,
			width:  w,
			height: h,
		}
	}
}
