package net

import (
	"bufio"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"netps/internal/socket"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func ParseSockets(pid int) ([]socket.Socket, error) {
	sockets := []socket.Socket{}
	inodes, err := getInodes(pid)
	if err != nil {
		return []socket.Socket{}, err
	}

	inodeSocketMap, errs := getInodeSocketMap()
	for _, e := range errs {
		slog.Error("ParseSockets() -> Error when parsing inode sockets map", "msg", e.Error())
	}

	for _, inode := range inodes {
		sockets = append(sockets, inodeSocketMap[inode])
	}
	return sockets, nil
}

func ParseSocketsByStates(pid int, state []socket.SocketState) ([]socket.Socket, error) {
	socks, err := ParseSockets(pid)
	if err != nil {
		return []socket.Socket{}, err
	}

	filtered := filterSockets(socks, state)
	return filtered, nil
}

func ParseRunningSockets() (map[int][]socket.Socket, error) {
	inodeSocketMap, errs := getInodeSocketMap()

	for _, e := range errs {
		slog.Error("ParseRunningSockets() -> Error when parsing inode sockets map", "msg", e.Error())
	}

	inodePID, err := mapInodeToPID()
	if err != nil {
		return nil, err
	}

	procMap := make(map[int][]socket.Socket)
	for inode, sock := range inodeSocketMap {
		if pid, ok := inodePID[inode]; ok {
			procMap[pid] = append(procMap[pid], sock)
		}
	}

	return procMap, nil
}

func parseProcNet(path string, proto string) (map[uint64]socket.Socket, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sockets := make(map[uint64]socket.Socket)

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return sockets, nil
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		state := parseState(fields[3])
		addr, port, err := parseHexAddr(fields[1])
		if err != nil {
			continue
		}

		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}

		sockets[inode] = socket.Socket{
			Proto: proto,
			Addr:  addr,
			Port:  port,
			State: state,
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

func parseState(state string) socket.SocketState {
	switch state {
	case "01":
		return socket.StateEstablished
	case "02":
		return socket.StateSynSent
	case "03":
		return socket.StateSynRecv
	case "04":
		return socket.StateFinWait1
	case "05":
		return socket.StateFinWait2
	case "06":
		return socket.StateTimeWait1
	case "07":
		return socket.StateClose
	case "08":
		return socket.StateCloseWait
	case "09":
		return socket.StateLastAck
	case "0A":
		return socket.StateListen
	case "0B":
		return socket.StateClosing
	case "0C":
		return socket.StateNewSynRecv
	default:
		return socket.StateUnknown
	}
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
			linkPath := filepath.Join(fdDir, fd.Name())
			link, err := os.Readlink(linkPath)
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

func getInodeSocketMap() (map[uint64]socket.Socket, []error) {
	inodeSocketMap := make(map[uint64]socket.Socket)
	errors := []error{}
	procNetfiles := []struct {
		path  string
		proto string
	}{
		{"/proc/net/tcp", "tcp"},
		{"/proc/net/tcp6", "tcp6"},
		{"/proc/net/udp", "udp"},
		{"/proc/net/udp6", "udp6"},
	}

	for _, f := range procNetfiles {
		m, err := parseProcNet(f.path, f.proto)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		maps.Copy(inodeSocketMap, m)
	}
	return inodeSocketMap, []error{}
}

func getInodes(pid int) ([]uint64, error) {

	result := []uint64{}
	fdDir := filepath.Join("/proc", strconv.Itoa(pid), "fd")
	fds, err := os.ReadDir(fdDir)
	if err != nil {
		return []uint64{}, err
	}

	for _, fd := range fds {
		linkPath := filepath.Join(fdDir, fd.Name())
		link, err := os.Readlink(linkPath)
		if err != nil {
			continue
		}

		if strings.HasPrefix(link, "socket:[") {
			inodeStr := strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")
			inode, err := strconv.ParseUint(inodeStr, 10, 64)
			if err == nil {
				result = append(result, inode)
			}
		}
	}
	return result, nil
}

func filterSockets(socks []socket.Socket, states []socket.SocketState) []socket.Socket {
	filtered := []socket.Socket{}
	for _, sock := range socks {
		if slices.Contains(states, sock.State) {
			filtered = append(filtered, sock)
		}
	}
	return filtered
}
