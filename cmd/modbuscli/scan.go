package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/edgeo/drivers/modbus"
	"github.com/spf13/cobra"
)

var (
	scanStartUnit uint8
	scanEndUnit   uint8
	scanStartAddr uint16
	scanEndAddr   uint16
	scanWorkers   int
	scanTimeout   time.Duration
	scanType      string
	scanNetwork   string
	scanPortStart int
	scanPortEnd   int
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for Modbus devices",
	Long: `Scan for Modbus devices on the network or detect active unit IDs.

Scan types:
  units     - Scan for active unit IDs on a single host (default)
  network   - Scan network range for Modbus devices
  registers - Scan address ranges to find used registers`,
	Example: `  # Scan for active unit IDs on a host
  modbuscli scan -H 192.168.1.100

  # Scan specific unit ID range
  modbuscli scan --start-unit 1 --end-unit 10 -H 192.168.1.100

  # Scan network for Modbus devices
  modbuscli scan --type network --network 192.168.1.0/24

  # Scan for used registers
  modbuscli scan --type registers -a 0 -e 1000 -H 192.168.1.100`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().Uint8Var(&scanStartUnit, "start-unit", 1, "Start unit ID for scanning")
	scanCmd.Flags().Uint8Var(&scanEndUnit, "end-unit", 247, "End unit ID for scanning")
	scanCmd.Flags().Uint16VarP(&scanStartAddr, "start-addr", "a", 0, "Start address for register scanning")
	scanCmd.Flags().Uint16VarP(&scanEndAddr, "end-addr", "e", 100, "End address for register scanning")
	scanCmd.Flags().IntVar(&scanWorkers, "workers", 10, "Number of concurrent workers")
	scanCmd.Flags().DurationVar(&scanTimeout, "scan-timeout", 1*time.Second, "Timeout for each scan attempt")
	scanCmd.Flags().StringVar(&scanType, "type", "units", "Scan type: units, network, registers")
	scanCmd.Flags().StringVar(&scanNetwork, "network", "", "Network CIDR for network scan (e.g., 192.168.1.0/24)")
	scanCmd.Flags().IntVar(&scanPortStart, "port-start", 502, "Start port for network scan")
	scanCmd.Flags().IntVar(&scanPortEnd, "port-end", 502, "End port for network scan")
}

type ScanResult struct {
	Address     string        `json:"address,omitempty"`
	UnitID      uint8         `json:"unit_id,omitempty"`
	StartAddr   uint16        `json:"start_addr,omitempty"`
	EndAddr     uint16        `json:"end_addr,omitempty"`
	Responsive  bool          `json:"responsive"`
	Error       string        `json:"error,omitempty"`
	ServerID    string        `json:"server_id,omitempty"`
	Latency     time.Duration `json:"latency_ms,omitempty"`
	RegisterQty int           `json:"register_qty,omitempty"`
}

func runScan(cmd *cobra.Command, args []string) error {
	switch scanType {
	case "units":
		return scanUnits()
	case "network":
		return scanNetworkDevices()
	case "registers":
		return scanRegisters()
	default:
		return fmt.Errorf("unknown scan type: %s", scanType)
	}
}

func scanUnits() error {
	outputInfo("Scanning unit IDs %d-%d on %s...", scanStartUnit, scanEndUnit, getAddress())

	results := make([]ScanResult, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, scanWorkers)

	for uid := scanStartUnit; uid <= scanEndUnit; uid++ {
		wg.Add(1)
		go func(unitID uint8) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := probeUnit(getAddress(), unitID)
			mu.Lock()
			if result.Responsive {
				results = append(results, result)
			}
			mu.Unlock()
		}(uid)
	}

	wg.Wait()

	// Sort by unit ID
	sort.Slice(results, func(i, j int) bool {
		return results[i].UnitID < results[j].UnitID
	})

	return outputScanResults("Unit Scan Results", results)
}

func probeUnit(addr string, unitID uint8) ScanResult {
	result := ScanResult{
		Address: addr,
		UnitID:  unitID,
	}

	client, err := modbus.NewClient(
		addr,
		modbus.WithUnitID(modbus.UnitID(unitID)),
		modbus.WithTimeout(scanTimeout),
		modbus.WithLogger(logger),
	)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		result.Error = err.Error()
		return result
	}

	start := time.Now()

	// Try to read a single holding register
	_, err = client.ReadHoldingRegisters(ctx, 0, 1)
	if err == nil {
		result.Responsive = true
		result.Latency = time.Since(start)
	} else {
		// Check if it's an exception (device responded but refused)
		result.Error = err.Error()
		// Modbus exceptions mean the device is responsive
		if strings.Contains(err.Error(), "exception") {
			result.Responsive = true
			result.Latency = time.Since(start)
		}
	}

	// Try to get server ID if responsive
	if result.Responsive {
		serverID, err := client.ReportServerID(ctx)
		if err == nil && len(serverID) > 0 {
			result.ServerID = strings.TrimRight(string(serverID), "\x00")
		}
	}

	return result
}

func scanNetworkDevices() error {
	if scanNetwork == "" {
		return fmt.Errorf("--network flag is required for network scan")
	}

	hosts, err := expandCIDR(scanNetwork)
	if err != nil {
		return fmt.Errorf("invalid network CIDR: %w", err)
	}

	outputInfo("Scanning %d hosts on network %s...", len(hosts), scanNetwork)

	results := make([]ScanResult, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, scanWorkers)

	for _, host := range hosts {
		for port := scanPortStart; port <= scanPortEnd; port++ {
			wg.Add(1)
			go func(h string, p int) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				addr := fmt.Sprintf("%s:%d", h, p)
				result := probeHost(addr)
				mu.Lock()
				if result.Responsive {
					results = append(results, result)
				}
				mu.Unlock()
			}(host, port)
		}
	}

	wg.Wait()

	// Sort by address
	sort.Slice(results, func(i, j int) bool {
		return results[i].Address < results[j].Address
	})

	return outputScanResults("Network Scan Results", results)
}

func probeHost(addr string) ScanResult {
	result := ScanResult{
		Address: addr,
		UnitID:  1,
	}

	// First check if port is open
	conn, err := net.DialTimeout("tcp", addr, scanTimeout)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	conn.Close()

	// Try Modbus communication
	return probeUnit(addr, 1)
}

func scanRegisters() error {
	outputInfo("Scanning registers %d-%d on %s (unit %d)...", scanStartAddr, scanEndAddr, getAddress(), unitID)

	client, err := createClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout*10)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	results := make([]ScanResult, 0)
	batchSize := uint16(10)

	for addr := scanStartAddr; addr <= scanEndAddr; {
		count := batchSize
		if addr+count > scanEndAddr+1 {
			count = scanEndAddr - addr + 1
		}

		readCtx, readCancel := context.WithTimeout(ctx, scanTimeout)
		_, err := client.ReadHoldingRegisters(readCtx, addr, count)
		readCancel()

		if err == nil {
			results = append(results, ScanResult{
				StartAddr:   addr,
				EndAddr:     addr + count - 1,
				Responsive:  true,
				RegisterQty: int(count),
			})
		} else if modbus.IsIllegalDataAddress(err) {
			// Address range not valid, skip
		} else {
			// Other error, might be partially valid
			for singleAddr := addr; singleAddr < addr+count && singleAddr <= scanEndAddr; singleAddr++ {
				singleCtx, singleCancel := context.WithTimeout(ctx, scanTimeout)
				_, err := client.ReadHoldingRegisters(singleCtx, singleAddr, 1)
				singleCancel()
				if err == nil {
					results = append(results, ScanResult{
						StartAddr:   singleAddr,
						EndAddr:     singleAddr,
						Responsive:  true,
						RegisterQty: 1,
					})
				}
			}
		}

		addr += count
	}

	// Merge contiguous ranges
	merged := mergeContiguousRanges(results)
	return outputRegisterScanResults(merged)
}

func mergeContiguousRanges(results []ScanResult) []ScanResult {
	if len(results) == 0 {
		return results
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].StartAddr < results[j].StartAddr
	})

	merged := []ScanResult{results[0]}
	for i := 1; i < len(results); i++ {
		last := &merged[len(merged)-1]
		current := results[i]

		if current.StartAddr <= last.EndAddr+1 {
			if current.EndAddr > last.EndAddr {
				last.EndAddr = current.EndAddr
			}
			last.RegisterQty = int(last.EndAddr - last.StartAddr + 1)
		} else {
			merged = append(merged, current)
		}
	}
	return merged
}

func expandCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Try as single IP
		if parsedIP := net.ParseIP(cidr); parsedIP != nil {
			return []string{parsedIP.String()}, nil
		}
		return nil, err
	}

	var hosts []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		hosts = append(hosts, ip.String())
	}

	// Remove network and broadcast addresses for /24 and smaller
	if len(hosts) > 2 {
		hosts = hosts[1 : len(hosts)-1]
	}

	return hosts, nil
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func outputScanResults(title string, results []ScanResult) error {
	if outputFmt == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	fmt.Printf("\n%s\n", color(colorBold, title))
	fmt.Println(strings.Repeat("-", 70))

	if len(results) == 0 {
		fmt.Println("No responsive devices found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ADDRESS\tUNIT ID\tLATENCY\tSERVER ID\tSTATUS")
	fmt.Fprintln(w, "-------\t-------\t-------\t---------\t------")

	for _, r := range results {
		latency := "-"
		if r.Latency > 0 {
			latency = fmt.Sprintf("%dms", r.Latency.Milliseconds())
		}
		serverID := r.ServerID
		if serverID == "" {
			serverID = "-"
		}
		status := color(colorGreen, "ONLINE")
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", r.Address, r.UnitID, latency, serverID, status)
	}
	w.Flush()

	fmt.Printf("\nFound %d responsive device(s)\n\n", len(results))
	return nil
}

func outputRegisterScanResults(results []ScanResult) error {
	if outputFmt == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	fmt.Printf("\n%s\n", color(colorBold, "Register Scan Results"))
	fmt.Println(strings.Repeat("-", 50))

	if len(results) == 0 {
		fmt.Println("No accessible registers found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "START ADDR\tEND ADDR\tCOUNT\tSTATUS")
	fmt.Fprintln(w, "----------\t--------\t-----\t------")

	total := 0
	for _, r := range results {
		status := color(colorGreen, "ACCESSIBLE")
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\n", r.StartAddr, r.EndAddr, r.RegisterQty, status)
		total += r.RegisterQty
	}
	w.Flush()

	fmt.Printf("\nFound %d accessible register(s) in %d range(s)\n\n", total, len(results))
	return nil
}
