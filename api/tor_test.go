package api

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestInspectTorFindsConfigAndReachablePort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	configPath := filepath.Join(t.TempDir(), "torrc")
	config := []byte(
		"SocksPort 127.0.0.1:" + strconv.Itoa(port) + "\n" +
			"ControlPort 9051\n" +
			"DataDirectory /var/lib/tor\n",
	)
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TORRC_PATH", configPath)

	status := inspectTor("127.0.0.1", port)
	if !status.Configured || status.ConfigPath != configPath {
		t.Fatalf("configuration not found: %#v", status)
	}
	if !status.Reachable {
		t.Fatalf("expected test SOCKS endpoint to be reachable: %#v", status)
	}
	if status.PortMatches == nil || !*status.PortMatches {
		t.Fatalf("expected configured port to match: %#v", status)
	}
	if status.DataDirectory != "/var/lib/tor" {
		t.Fatalf("unexpected data directory: %q", status.DataDirectory)
	}
}
