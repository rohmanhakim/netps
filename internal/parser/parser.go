package parser

import (
	"bufio"
	"fmt"
	"maps"
	"net"
	"netps/pkg/model"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func parseProcNet(path string, proto string) (map[uint64]model.SocketInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sockets := make(map[uint64]model.SocketInfo)

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return sockets, nil
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		state := fields[3]
		if state != "0A" {
			continue
		}
		addr, port, err := parseHexAddr(fields[1])
		if err != nil {
			continue
		}

		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}

		sockets[inode] = model.SocketInfo{
			Proto: proto,
			Addr:  addr,
			Port:  port,
		}
	}

	return sockets, nil
}

func parseHexAddr(s string) (string, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid addr")
	}

	port64, err := strconv.ParseUint(parts[1], 16, 16)
	if err != nil {
		return "", 0, err
	}

	ipHex := parts[0]
	var ip net.IP

	if len(ipHex) == 8 { // IPv4
		b := make([]byte, 4)
		for i := 0; i < 4; i++ {
			v, _ := strconv.ParseUint(ipHex[i*2:i*2+2], 16, 8)
			b[3-i] = byte(v)
		}
		ip = net.IP(b)
	} else { // IPv6
		b := make([]byte, 16)
		for i := 0; i < 16; i++ {
			v, _ := strconv.ParseUint(ipHex[i*2:i*2+2], 16, 8)
			b[15-i] = byte(v)
		}
		ip = net.IP(b)
	}

	return ip.String(), int(port64), nil
}

func mapInodeToPID() (map[uint64]int, error) {
	result := make(map[uint64]int)

	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	for _, e := range procEntries {
		if !e.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		fdDir := filepath.Join("/proc", e.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}

			if strings.HasPrefix(link, "socket:[") {
				inodeStr := strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")
				inode, err := strconv.ParseUint(inodeStr, 10, 64)
				if err == nil {
					result[inode] = pid
				}
			}
		}
	}

	return result, nil
}

func ScanListeningPortsProcfs() ([]model.Process, error) {
	socketMaps := make(map[uint64]model.SocketInfo)

	files := []struct {
		path  string
		proto string
	}{
		{"/proc/net/tcp", "tcp"},
		{"/proc/net/tcp6", "tcp6"},
		{"/proc/net/udp", "udp"},
		{"/proc/net/udp6", "udp6"},
	}

	for _, f := range files {
		m, err := parseProcNet(f.path, f.proto)
		if err == nil {
			maps.Copy(socketMaps, m)
		}
	}

	inodePID, err := mapInodeToPID()
	if err != nil {
		return nil, err
	}

	out := []model.Process{}
	for inode, sock := range socketMaps {
		if pid, ok := inodePID[inode]; ok {
			proc := model.Process{
				Name:       "",
				PID:        pid,
				SocketInfo: sock,
			}
			out = append(out, proc)
		}
	}

	return out, nil
}
