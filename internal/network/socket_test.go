package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestListenAndPort(t *testing.T) {
	root := t.TempDir()
	writeProcTCP(t, root, "tcp", "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"+
		"   0: 00000000:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 12345\n"+
		"   1: 0100007F:0CEA 00000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 12346\n"+
		"   2: 00000000:01BB 0100007F:9C40 01 00000000:00000000 00:00000000 00000000 0 0 12347\n")
	writeProcTCP(t, root, "tcp6", "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"+
		"   0: 00000000000000000000000000000000:1F90 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 22345\n"+
		"   1: 00000000000000000000000001000000:0050 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 22346\n")

	reader := NewReader(root)
	listeners, err := reader.Listen()
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	if len(listeners) != 4 {
		t.Fatalf("Listen() returned %d listeners, want 4: %#v", len(listeners), listeners)
	}
	if listeners[0].LocalAddress != "0.0.0.0" || listeners[0].LocalPort != 80 ||
		listeners[0].State != "LISTEN" || listeners[0].RemoteAddress != "0.0.0.0" ||
		listeners[0].RemotePort != 0 || listeners[0].AddressType != "IPv4" ||
		listeners[0].Inode != 12345 {
		t.Errorf("first listener = %#v", listeners[0])
	}
	var foundIPv6 bool
	for _, socket := range listeners {
		if socket.LocalAddress == "::" && socket.LocalPort == 8080 &&
			socket.RemoteAddress == "::" && socket.RemotePort == 0 &&
			socket.AddressType == "IPv6" && socket.Inode == 22345 {
			foundIPv6 = true
		}
	}
	if !foundIPv6 {
		t.Errorf("Listen() did not include the expected IPv6 listener: %#v", listeners)
	}

	matches, err := reader.Port(8080)
	if err != nil {
		t.Fatalf("Port(8080) error = %v", err)
	}
	if len(matches) != 1 || matches[0].LocalAddress != "::" {
		t.Fatalf("Port(8080) = %#v, want the IPv6 listener", matches)
	}
}

func TestPortAssociatesAllProcessesWithSocketInode(t *testing.T) {
	netRoot := t.TempDir()
	procRoot := t.TempDir()
	writeProcTCP(t, netRoot, "tcp", "header\n"+
		"0: 00000000:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 12345\n")
	writeProcTCP(t, netRoot, "tcp6", "header\n")
	writeProcessFD(t, procRoot, 1520, "java", "java -jar application.jar", 12345)
	writeProcessFD(t, procRoot, 1521, "worker", "/usr/bin/worker", 12345)

	sockets, err := NewReaderWithProcRoot(netRoot, procRoot).Port(8080)
	if err != nil {
		t.Fatalf("Port(8080) error = %v", err)
	}
	if len(sockets) != 1 || len(sockets[0].Processes) != 2 {
		t.Fatalf("Port(8080) process associations = %#v", sockets)
	}
	first := sockets[0].Processes[0]
	if first.PID != 1520 || first.Name != "java" || first.User != "root" ||
		first.Command != "java -jar application.jar" {
		t.Errorf("first process = %#v", first)
	}
}

func TestParseLocalAddress(t *testing.T) {
	tests := []struct {
		input string
		ipv6  bool
		want  string
		port  uint16
	}{
		{"0100007F:1F90", false, "127.0.0.1", 8080},
		{"00000000000000000000000001000000:0050", true, "::1", 80},
	}
	for _, test := range tests {
		address, port, err := parseProcAddress(test.input, test.ipv6)
		if err != nil {
			t.Errorf("parseLocalAddress(%q) error = %v", test.input, err)
			continue
		}
		if address != test.want || port != test.port {
			t.Errorf("parseLocalAddress(%q) = (%q, %d), want (%q, %d)",
				test.input, address, port, test.want, test.port)
		}
	}
}

func TestReadTCPFileRejectsMalformedRows(t *testing.T) {
	root := t.TempDir()
	writeProcTCP(t, root, "tcp", "header\nmalformed\n")
	reader := NewReader(root)
	if _, err := reader.readTCPFile("tcp"); err == nil {
		t.Fatal("readTCPFile() error = nil, want malformed-row error")
	}
}

func TestReadTCPFileRejectsInvalidInode(t *testing.T) {
	root := t.TempDir()
	writeProcTCP(t, root, "tcp", "header\n"+
		"0: 00000000:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 invalid\n")
	if _, err := NewReader(root).readTCPFile("tcp"); err == nil {
		t.Fatal("readTCPFile() error = nil, want invalid inode error")
	}
}

func writeProcessFD(t *testing.T, root string, pid int, name, command string, inode uint64) {
	t.Helper()
	dir := filepath.Join(root, strconv.Itoa(pid))
	if err := os.MkdirAll(filepath.Join(dir, "fd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(fmt.Sprintf("socket:[%d]", inode), filepath.Join(dir, "fd", "3")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "comm"), []byte(name+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "status"), []byte("Uid:\t0\t0\t0\t0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(strings.ReplaceAll(command, " ", "\x00")+"\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeProcTCP(t *testing.T, root, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
