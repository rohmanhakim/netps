package procfs

import (
	"context"
	"netps/internal/process"
	"netps/internal/procfs/cmdline"
	"netps/internal/procfs/comm"
	"netps/internal/procfs/exe"
	"netps/internal/procfs/net"
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
	detail := process.ProcessDetail{
		ExecPath: execPath,
		Command:  command,
	}
	return detail, nil
}
