package api

import (
	"bufio"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type TorStatus struct {
	Configured    bool      `json:"configured"`
	Reachable     bool      `json:"reachable"`
	Host          string    `json:"host"`
	Port          int       `json:"port"`
	ConfigPath    string    `json:"configPath,omitempty"`
	SOCKSPorts    []string  `json:"socksPorts"`
	ControlPorts  []string  `json:"controlPorts"`
	DataDirectory string    `json:"dataDirectory,omitempty"`
	TorBinary     string    `json:"torBinary,omitempty"`
	PortMatches   *bool     `json:"portMatches,omitempty"`
	SearchedPaths []string  `json:"searchedPaths"`
	CheckedAt     time.Time `json:"checkedAt"`
}

func torConfigCandidates() []string {
	candidates := []string{}
	if configured := strings.TrimSpace(os.Getenv("TORRC_PATH")); configured != "" {
		candidates = append(candidates, configured)
	}

	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			candidates = append(candidates, filepath.Join(appData, "tor", "torrc"))
		}
		if home != "" {
			candidates = append(candidates,
				filepath.Join(home, "Desktop", "Tor Browser", "Browser", "TorBrowser", "Data", "Tor", "torrc"),
			)
		}
	case "darwin":
		candidates = append(candidates,
			"/opt/homebrew/etc/tor/torrc",
			"/usr/local/etc/tor/torrc",
		)
		if home != "" {
			candidates = append(candidates,
				filepath.Join(home, ".torrc"),
				filepath.Join(home, "Library", "Application Support", "TorBrowser-Data", "Tor", "torrc"),
			)
		}
	default:
		candidates = append(candidates,
			"/etc/tor/torrc",
			"/usr/local/etc/tor/torrc",
		)
		if home != "" {
			candidates = append(candidates,
				filepath.Join(home, ".torrc"),
				filepath.Join(home, ".config", "tor", "torrc"),
			)
		}
	}

	seen := map[string]struct{}{}
	unique := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}
	return unique
}

func readTorConfig(path string, status *TorStatus) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	status.Configured = true
	status.ConfigPath = path
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value := strings.Join(fields[1:], " ")
		switch strings.ToLower(fields[0]) {
		case "socksport":
			status.SOCKSPorts = append(status.SOCKSPorts, value)
		case "controlport":
			status.ControlPorts = append(status.ControlPorts, value)
		case "datadirectory":
			status.DataDirectory = value
		}
	}
}

func configuredPortMatches(values []string, requestedPort int) *bool {
	if len(values) == 0 {
		return nil
	}
	match := false
	foundNumericPort := false
	for _, value := range values {
		value = strings.Fields(value)[0]
		if strings.EqualFold(value, "auto") || strings.HasPrefix(value, "unix:") {
			continue
		}
		portValue := value
		if _, port, err := net.SplitHostPort(value); err == nil {
			portValue = port
		} else if separator := strings.LastIndex(value, ":"); separator >= 0 {
			portValue = value[separator+1:]
		}
		port, err := strconv.Atoi(portValue)
		if err != nil {
			continue
		}
		foundNumericPort = true
		if port == requestedPort {
			match = true
		}
	}
	if !foundNumericPort {
		return nil
	}
	return &match
}

func inspectTor(host string, port int) TorStatus {
	status := TorStatus{
		Host:          host,
		Port:          port,
		SOCKSPorts:    []string{},
		ControlPorts:  []string{},
		SearchedPaths: torConfigCandidates(),
		CheckedAt:     time.Now().UTC(),
	}
	for _, candidate := range status.SearchedPaths {
		if fileInfo, err := os.Stat(candidate); err == nil && !fileInfo.IsDir() {
			readTorConfig(candidate, &status)
			break
		}
	}
	status.PortMatches = configuredPortMatches(status.SOCKSPorts, port)
	if binary, err := exec.LookPath("tor"); err == nil {
		status.TorBinary = binary
	}

	connection, err := net.DialTimeout(
		"tcp",
		net.JoinHostPort(host, strconv.Itoa(port)),
		800*time.Millisecond,
	)
	if err == nil {
		status.Reachable = true
		_ = connection.Close()
	}
	return status
}

func (s *Server) handleTorStatus(writer http.ResponseWriter, request *http.Request) {
	host := strings.TrimSpace(request.URL.Query().Get("host"))
	if host == "" {
		host = "127.0.0.1"
	}
	if len(host) > 255 {
		writeAPIError(writer, http.StatusBadRequest, "invalid Tor proxy host")
		return
	}

	port := 9050
	if input := request.URL.Query().Get("port"); input != "" {
		parsed, err := strconv.Atoi(input)
		if err != nil || parsed < 1 || parsed > 65535 {
			writeAPIError(writer, http.StatusBadRequest, "Tor proxy port must be between 1 and 65535")
			return
		}
		port = parsed
	}
	writeJSON(writer, http.StatusOK, inspectTor(host, port))
}
