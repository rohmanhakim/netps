package procfs

import (
	"context"
	"netps/internal/process"
	"netps/internal/procfs/cmdline"
	"netps/internal/procfs/comm"
	"netps/internal/procfs/exe"
	"netps/internal/procfs/net"
	"netps/internal/procfs/stat"
	"netps/internal/procfs/status"
	"netps/internal/procfs/uptime"
	"netps/internal/socket"
	"os/user"
	"strconv"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (p *Client) ListRunnings(ctx context.Context) ([]process.ProcessSummary, error) {
	runningSockets, err := net.ParseRunningSockets()
	if err != nil {
		return nil, err
	}

	out := []process.ProcessSummary{}
	for pid, sockets := range runningSockets {
		name, err := comm.ParseProcessName(pid)
		if err != nil {
			return nil, err
		}
		proc := process.NewSummary(pid, name).
			WithAggregatedSockets(sockets).
			WithFilteredListenPorts(sockets)

		out = append(out, *proc)
	}
	return out, nil
}

func (p *Client) Detail(ctx context.Context, pid int) (process.ProcessDetail, error) {
	execPath, err := exe.ParseProcExe(pid)
	if err != nil {
		return process.ProcessDetail{}, err
	}
	command, err := cmdline.ParseCmdLine(pid)
	if err != nil {
		return process.ProcessDetail{}, err
	}

	processStat, err := stat.ParseStat(pid)
	if err != nil {
		return process.ProcessDetail{}, nil
	}
	ppid := processStat.PPID

	parentName, err := comm.ParseProcessName(ppid)
	if err != nil {
		return process.ProcessDetail{}, nil
	}

	detail := process.ProcessDetail{
		ExecPath:   execPath,
		Command:    command,
		PPID:       ppid,
		ParentName: parentName,
	}

	return detail, nil
}

func (p *Client) UpTime(ctx context.Context) (float64, error) {
	upTime, err := uptime.ParseSystemUptime()
	if err != nil {
		return -1, err
	}
	return upTime, nil
}

func (p *Client) Resource(ctx context.Context, pid int) (process.ProcessResource, error) {
	processStat, err := stat.ParseStat(pid)
	if err != nil {
		return process.ProcessResource{}, err
	}

	resource := process.ProcessResource{
		ResidentSetSizePage:    processStat.RSS,
		StartTimeTick:          processStat.StartTime,
		VirtualMemorySize:      processStat.VSize,
		UserCPUTimeClockTick:   processStat.UTime,
		SystemCPUTimeClockTick: processStat.STime,
	}
	return resource, nil
}

func (s *Client) User(ctx context.Context, pid int) (process.ProcessUser, error) {
	realId, err := status.ParseRealUID(pid)
	if err != nil {
		return process.ProcessUser{}, err
	}

	u, err := user.LookupId(strconv.Itoa(realId))
	if err != nil {
		return process.ProcessUser{}, err
	}

	effectiveId, err := status.ParseEffectiveUID(pid)
	if err != nil {
		return process.ProcessUser{}, err
	}

	user := process.ProcessUser{
		RealUID:    realId,
		Name:       u.Username,
		Privileged: effectiveId == 0,
	}
	return user, nil
}

func (s *Client) SocketsByStates(ctx context.Context, pid int, states []socket.SocketState) ([]socket.Socket, error) {
	sockets, err := net.ParseSocketsByStates(pid, states)
	if err != nil {
		return []socket.Socket{}, err
	}
	return sockets, nil
}
