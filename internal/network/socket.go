package network

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Socket struct {
	LocalAddress string
	LocalPort    uint16
	Protocol     string
	State        string
}

type Reader struct {
	root string
}

func NewReader(procNetRoot string) *Reader {
	if procNetRoot == "" {
		procNetRoot = "/proc/net"
	}
	return &Reader{root: procNetRoot}
}

func (r *Reader) readTCPFile(name string) ([]Socket, error) {
	path := filepath.Join(r.root, name)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		return nil, fmt.Errorf("read %s: missing header", path)
	}

	sockets := make([]Socket, 0)
	lineNumber := 1
	for scanner.Scan() {
		lineNumber++
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			return nil, fmt.Errorf("parse %s line %d: expected at least 4 fields", path, lineNumber)
		}

		address, port, err := parseLocalAddress(fields[1], name == "tcp6")
		if err != nil {
			return nil, fmt.Errorf("parse %s line %d: %w", path, lineNumber, err)
		}
		state, ok := tcpState(fields[3])
		if !ok {
			return nil, fmt.Errorf("parse %s line %d: invalid TCP state %q", path, lineNumber, fields[3])
		}

		sockets = append(sockets, Socket{
			LocalAddress: address,
			LocalPort:    port,
			Protocol:     "TCP",
			State:        state,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return sockets, nil
}

func parseLocalAddress(value string, ipv6 bool) (string, uint16, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid local address %q", value)
	}

	expectedLength := 8
	if ipv6 {
		expectedLength = 32
	}
	if len(parts[0]) != expectedLength {
		return "", 0, fmt.Errorf("invalid local address %q", value)
	}
	if len(parts[1]) != 4 {
		return "", 0, fmt.Errorf("invalid local port in %q", value)
	}
	raw, err := hex.DecodeString(parts[0])
	if err != nil {
		return "", 0, fmt.Errorf("decode local address %q: %w", value, err)
	}
	port, err := strconv.ParseUint(parts[1], 16, 16)
	if err != nil {
		return "", 0, fmt.Errorf("decode local port %q: %w", parts[1], err)
	}

	address := make([]byte, len(raw))
	if ipv6 {
		for i := 0; i < len(raw); i += 4 {
			word := binary.BigEndian.Uint32(raw[i : i+4])
			binary.NativeEndian.PutUint32(address[i:i+4], word)
		}
	} else {
		word := binary.BigEndian.Uint32(raw)
		binary.NativeEndian.PutUint32(address, word)
	}
	return net.IP(address).String(), uint16(port), nil
}

func tcpState(value string) (string, bool) {
	value = strings.ToUpper(value)
	if len(value) != 2 {
		return "", false
	}
	switch value {
	case "00":
		return "UNKNOWN", true
	case "01":
		return "ESTABLISHED", true
	case "02":
		return "SYN_SENT", true
	case "03":
		return "SYN_RECV", true
	case "04":
		return "FIN_WAIT1", true
	case "05":
		return "FIN_WAIT2", true
	case "06":
		return "TIME_WAIT", true
	case "07":
		return "CLOSE", true
	case "08":
		return "CLOSE_WAIT", true
	case "09":
		return "LAST_ACK", true
	case "0A":
		return "LISTEN", true
	case "0B":
		return "CLOSING", true
	case "0C":
		return "NEW_SYN_RECV", true
	default:
		if _, err := strconv.ParseUint(value, 16, 8); err != nil {
			return "", false
		}
		return "UNKNOWN(" + value + ")", true
	}
}
