package probe

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// NetworkConnection represents a network connection
type NetworkConnection struct {
	Protocol    string `json:"protocol"`
	LocalAddr   string `json:"local_addr"`
	LocalPort   uint16 `json:"local_port"`
	RemoteAddr  string `json:"remote_addr"`
	RemotePort  uint16 `json:"remote_port"`
	State       string `json:"state"`
	PID         int    `json:"pid,omitempty"`
	ProcessName string `json:"process_name,omitempty"`
}

// NetworkInfo contains aggregated network information
type NetworkInfo struct {
	Connections      []NetworkConnection `json:"connections"`
	TotalConnections int                 `json:"total_connections"`
	TCPConnections   int                 `json:"tcp_connections"`
	UDPConnections   int                 `json:"udp_connections"`
	ListeningPorts   []ListeningPort     `json:"listening_ports"`
	Timestamp        time.Time           `json:"timestamp"`
}

// ListeningPort represents a listening port
type ListeningPort struct {
	Protocol    string `json:"protocol"`
	Port        uint16 `json:"port"`
	Addr        string `json:"addr"`
	PID         int    `json:"pid,omitempty"`
	ProcessName string `json:"process_name,omitempty"`
}

// NetworkCollector collects network connection information
type NetworkCollector struct {
	procPath         string
	includeLocalhost bool
	maxConnections   int
	resolveProcesses bool
}

// NewNetworkCollector creates a new network collector
func NewNetworkCollector(includeLocalhost bool, maxConnections int, resolveProcesses bool) *NetworkCollector {
	return &NetworkCollector{
		procPath:         "/proc",
		includeLocalhost: includeLocalhost,
		maxConnections:   maxConnections,
		resolveProcesses: resolveProcesses,
	}
}

// NewNetworkCollectorWithPath creates a network collector with custom proc path (for testing)
func NewNetworkCollectorWithPath(procPath string, includeLocalhost bool, maxConnections int, resolveProcesses bool) *NetworkCollector {
	return &NetworkCollector{
		procPath:         procPath,
		includeLocalhost: includeLocalhost,
		maxConnections:   maxConnections,
		resolveProcesses: resolveProcesses,
	}
}

// Collect gathers network connection information
func (n *NetworkCollector) Collect() (*NetworkInfo, error) {
	info := &NetworkInfo{
		Timestamp:      time.Now(),
		Connections:    make([]NetworkConnection, 0),
		ListeningPorts: make([]ListeningPort, 0),
	}

	// Build inode to PID mapping if process resolution is enabled
	var inodeToPID map[string]int
	var inodeToName map[string]string
	if n.resolveProcesses {
		inodeToPID, inodeToName = n.buildInodeMapping()
	}

	// Collect TCP connections
	tcpConns, err := n.collectTCPConnections(inodeToPID, inodeToName)
	if err != nil {
		return nil, fmt.Errorf("failed to collect TCP connections: %w", err)
	}
	info.Connections = append(info.Connections, tcpConns...)
	info.TCPConnections = len(tcpConns)

	// Collect UDP connections
	udpConns, err := n.collectUDPConnections(inodeToPID, inodeToName)
	if err != nil {
		return nil, fmt.Errorf("failed to collect UDP connections: %w", err)
	}
	info.Connections = append(info.Connections, udpConns...)
	info.UDPConnections = len(udpConns)

	info.TotalConnections = len(info.Connections)

	// Extract listening ports
	info.ListeningPorts = n.extractListeningPorts(info.Connections)

	return info, nil
}

// collectTCPConnections reads TCP connections from /proc/net/tcp
func (n *NetworkCollector) collectTCPConnections(inodeToPID map[string]int, inodeToName map[string]string) ([]NetworkConnection, error) {
	return n.parseNetFile(fmt.Sprintf("%s/net/tcp", n.procPath), "tcp", inodeToPID, inodeToName)
}

// collectUDPConnections reads UDP connections from /proc/net/udp
func (n *NetworkCollector) collectUDPConnections(inodeToPID map[string]int, inodeToName map[string]string) ([]NetworkConnection, error) {
	return n.parseNetFile(fmt.Sprintf("%s/net/udp", n.procPath), "udp", inodeToPID, inodeToName)
}

// parseNetFile parses /proc/net/tcp or /proc/net/udp
func (n *NetworkCollector) parseNetFile(path, protocol string, inodeToPID map[string]int, inodeToName map[string]string) ([]NetworkConnection, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var connections []NetworkConnection
	scanner := bufio.NewScanner(file)

	// Skip header line
	scanner.Scan()

	count := 0
	for scanner.Scan() {
		if n.maxConnections > 0 && count >= n.maxConnections {
			break
		}

		line := scanner.Text()
		conn, err := n.parseConnectionLine(line, protocol)
		if err != nil {
			continue
		}

		// Skip localhost connections if not included
		if !n.includeLocalhost && (conn.LocalAddr == "127.0.0.1" || conn.RemoteAddr == "127.0.0.1") {
			continue
		}

		// Resolve process if mapping is available
		if inodeToPID != nil {
			// Extract inode from line (last field)
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				inode := fields[9]
				if pid, ok := inodeToPID[inode]; ok {
					conn.PID = pid
					if name, ok := inodeToName[inode]; ok {
						conn.ProcessName = name
					}
				}
			}
		}

		connections = append(connections, *conn)
		count++
	}

	return connections, scanner.Err()
}

// parseConnectionLine parses a line from /proc/net/tcp or /proc/net/udp
func (n *NetworkCollector) parseConnectionLine(line, protocol string) (*NetworkConnection, error) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return nil, fmt.Errorf("invalid line format")
	}

	conn := &NetworkConnection{
		Protocol: protocol,
	}

	// Parse local address
	localParts := strings.Split(fields[1], ":")
	if len(localParts) != 2 {
		return nil, fmt.Errorf("invalid local address format")
	}
	conn.LocalAddr = hexToIP(localParts[0])
	port, _ := strconv.ParseUint(localParts[1], 16, 16)
	conn.LocalPort = uint16(port)

	// Parse remote address
	remoteParts := strings.Split(fields[2], ":")
	if len(remoteParts) != 2 {
		return nil, fmt.Errorf("invalid remote address format")
	}
	conn.RemoteAddr = hexToIP(remoteParts[0])
	port, _ = strconv.ParseUint(remoteParts[1], 16, 16)
	conn.RemotePort = uint16(port)

	// Parse state
	if protocol == "tcp" {
		stateHex := fields[3]
		state, _ := strconv.ParseUint(stateHex, 16, 8)
		conn.State = tcpStateToString(uint8(state))
	} else {
		conn.State = "ACTIVE"
	}

	return conn, nil
}

// hexToIP converts hex IP address to dotted decimal notation
func hexToIP(hexIP string) string {
	if len(hexIP) != 8 {
		return hexIP
	}

	// Reverse byte order (little-endian)
	var octets [4]string
	for i := 0; i < 4; i++ {
		octet, _ := strconv.ParseUint(hexIP[i*2:i*2+2], 16, 8)
		octets[3-i] = strconv.Itoa(int(octet))
	}

	return strings.Join(octets[:], ".")
}

// tcpStateToString converts TCP state number to string
func tcpStateToString(state uint8) string {
	states := map[uint8]string{
		0x01: "ESTABLISHED",
		0x02: "SYN_SENT",
		0x03: "SYN_RECV",
		0x04: "FIN_WAIT1",
		0x05: "FIN_WAIT2",
		0x06: "TIME_WAIT",
		0x07: "CLOSE",
		0x08: "CLOSE_WAIT",
		0x09: "LAST_ACK",
		0x0A: "LISTEN",
		0x0B: "CLOSING",
	}

	if s, ok := states[state]; ok {
		return s
	}
	return fmt.Sprintf("UNKNOWN(%02X)", state)
}

// buildInodeMapping creates a mapping from socket inodes to PIDs and process names
func (n *NetworkCollector) buildInodeMapping() (map[string]int, map[string]string) {
	inodeToPID := make(map[string]int)
	inodeToName := make(map[string]string)

	entries, err := os.ReadDir(n.procPath)
	if err != nil {
		return inodeToPID, inodeToName
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// Get process name
		statPath := fmt.Sprintf("%s/%d/stat", n.procPath, pid)
		statData, err := os.ReadFile(statPath)
		if err != nil {
			continue
		}

		processName := extractProcessName(string(statData))

		// Read socket inodes from /proc/[pid]/fd
		fdPath := fmt.Sprintf("%s/%d/fd", n.procPath, pid)
		fdEntries, err := os.ReadDir(fdPath)
		if err != nil {
			continue
		}

		for _, fdEntry := range fdEntries {
			link, err := os.Readlink(fmt.Sprintf("%s/%s", fdPath, fdEntry.Name()))
			if err != nil {
				continue
			}

			// Check if it's a socket
			if strings.HasPrefix(link, "socket:[") {
				inode := strings.TrimPrefix(link, "socket:[")
				inode = strings.TrimSuffix(inode, "]")
				inodeToPID[inode] = pid
				inodeToName[inode] = processName
			}
		}
	}

	return inodeToPID, inodeToName
}

// extractProcessName extracts process name from /proc/[pid]/stat
func extractProcessName(stat string) string {
	startIdx := strings.IndexByte(stat, '(')
	endIdx := strings.LastIndexByte(stat, ')')

	if startIdx == -1 || endIdx == -1 {
		return ""
	}

	return stat[startIdx+1 : endIdx]
}

// extractListeningPorts extracts listening ports from connections
func (n *NetworkCollector) extractListeningPorts(connections []NetworkConnection) []ListeningPort {
	var ports []ListeningPort
	seen := make(map[string]bool)

	for _, conn := range connections {
		if conn.State == "LISTEN" {
			key := fmt.Sprintf("%s:%d", conn.Protocol, conn.LocalPort)
			if !seen[key] {
				ports = append(ports, ListeningPort{
					Protocol:    conn.Protocol,
					Port:        conn.LocalPort,
					Addr:        conn.LocalAddr,
					PID:         conn.PID,
					ProcessName: conn.ProcessName,
				})
				seen[key] = true
			}
		}
	}

	return ports
}
