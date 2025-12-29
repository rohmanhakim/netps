package procfs

import (
	"context"
	"netps/internal/process"
	"netps/internal/procfs/cmdline"
	"netps/internal/procfs/comm"
	"netps/internal/procfs/exe"
	"netps/internal/procfs/net"
	"netps/internal/procfs/stat"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (p *Client) ListRunnings(ctx context.Context) ([]process.Process, error) {
	runningSockets, err := net.ParseRunningSockets()
	if err != nil {
		return nil, err
	}

	out := []process.Process{}
	for pid, sockets := range runningSockets {
		name, err := comm.ParseProcessName(pid)
		if err != nil {
			return nil, err
		}
		proc := process.NewProcess(pid, name).WithSockets(sockets)
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
