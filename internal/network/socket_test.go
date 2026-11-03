package network

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListenAndPort(t *testing.T) {
	root := t.TempDir()
	writeProcTCP(t, root, "tcp", "  sl  local_address rem_address   st\n"+
		"   0: 00000000:0050 00000000:0000 0A\n"+
		"   1: 0100007F:0CEA 00000000:0000 0A\n"+
		"   2: 00000000:01BB 0100007F:9C40 01\n")
	writeProcTCP(t, root, "tcp6", "  sl  local_address rem_address   st\n"+
		"   0: 00000000000000000000000000000000:1F90 00000000000000000000000000000000:0000 0A\n"+
		"   1: 00000000000000000000000001000000:0050 00000000000000000000000000000000:0000 0A\n")

	reader := NewReader(root)
	listeners, err := reader.Listen()
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	if len(listeners) != 4 {
		t.Fatalf("Listen() returned %d listeners, want 4: %#v", len(listeners), listeners)
	}
	if listeners[0].LocalAddress != "0.0.0.0" || listeners[0].LocalPort != 80 || listeners[0].State != "LISTEN" {
		t.Errorf("first listener = %#v", listeners[0])
	}
	var foundIPv6 bool
	for _, socket := range listeners {
		if socket.LocalAddress == "::" && socket.LocalPort == 8080 {
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
		address, port, err := parseLocalAddress(test.input, test.ipv6)
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

func writeProcTCP(t *testing.T, root, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
