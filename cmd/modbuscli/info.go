package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/edgeo/drivers/modbus"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"probe", "ping"},
	Short:   "Get device information",
	Long: `Probe a Modbus device and retrieve all available information.

This command attempts to:
  - Test connectivity
  - Read server identification (FC17)
  - Read exception status (FC07)
  - Measure response latency`,
	Example: `  modbuscli info -H 192.168.1.100
  modbuscli info -H 10.0.0.50 -u 2`,
	RunE: runInfo,
}

type DeviceInfo struct {
	Address      string        `json:"address"`
	UnitID       uint8         `json:"unit_id"`
	Reachable    bool          `json:"reachable"`
	Connected    bool          `json:"connected"`
	Latency      time.Duration `json:"latency_ms"`
	ServerID     string        `json:"server_id,omitempty"`
	ServerIDHex  string        `json:"server_id_hex,omitempty"`
	RunStatus    string        `json:"run_status,omitempty"`
	Exception    uint8         `json:"exception_status,omitempty"`
	Capabilities []string      `json:"capabilities,omitempty"`
	Error        string        `json:"error,omitempty"`
}

func runInfo(cmd *cobra.Command, args []string) error {
	info := DeviceInfo{
		Address: getAddress(),
		UnitID:  unitID,
	}

	conn, err := net.DialTimeout("tcp", info.Address, timeout)
	if err != nil {
		info.Error = err.Error()
		return outputDeviceInfo(&info)
	}
	conn.Close()
	info.Reachable = true

	client, err := modbus.NewClient(
		info.Address,
		modbus.WithUnitID(modbus.UnitID(unitID)),
		modbus.WithTimeout(timeout),
		modbus.WithLogger(logger),
	)
	if err != nil {
		info.Error = err.Error()
		return outputDeviceInfo(&info)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		info.Error = err.Error()
		return outputDeviceInfo(&info)
	}
	info.Connected = true

	start := time.Now()
	_, err = client.ReadHoldingRegisters(ctx, 0, 1)
	info.Latency = time.Since(start)

	if err != nil && !strings.Contains(err.Error(), "exception") {
		info.Error = err.Error()
	}

	serverID, err := client.ReportServerID(ctx)
	if err == nil && len(serverID) > 0 {
		info.ServerIDHex = fmt.Sprintf("% X", serverID)
		if isPrintable(serverID) {
			info.ServerID = strings.TrimRight(string(serverID), "\x00")
		}
		if len(serverID) >= 2 {
			if serverID[1] == 0xFF {
				info.RunStatus = "Running"
			} else {
				info.RunStatus = "Stopped"
			}
		}
	}

	exStatus, err := client.ReadExceptionStatus(ctx)
	if err == nil {
		info.Exception = exStatus
	}

	info.Capabilities = testCapabilities(ctx, client)

	return outputDeviceInfo(&info)
}

func testCapabilities(ctx context.Context, client *modbus.Client) []string {
	var caps []string

	tests := []struct {
		name string
		test func() error
	}{
		{"Read Coils (FC01)", func() error {
			_, err := client.ReadCoils(ctx, 0, 1)
			return err
		}},
		{"Read Discrete Inputs (FC02)", func() error {
			_, err := client.ReadDiscreteInputs(ctx, 0, 1)
			return err
		}},
		{"Read Holding Registers (FC03)", func() error {
			_, err := client.ReadHoldingRegisters(ctx, 0, 1)
			return err
		}},
		{"Read Input Registers (FC04)", func() error {
			_, err := client.ReadInputRegisters(ctx, 0, 1)
			return err
		}},
	}

	for _, t := range tests {
		err := t.test()
		if err == nil {
			caps = append(caps, t.name)
		} else if strings.Contains(err.Error(), "illegal data address") {
			caps = append(caps, t.name+" (no data at addr 0)")
		}
	}

	return caps
}

func outputDeviceInfo(info *DeviceInfo) error {
	if outputFmt == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	}

	fmt.Println()
	fmt.Println(color(colorBold, "Device Information"))
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("Address:      %s\n", info.Address)
	fmt.Printf("Unit ID:      %d\n", info.UnitID)

	if info.Reachable {
		fmt.Printf("TCP:          %s\n", color(colorGreen, "Reachable"))
	} else {
		fmt.Printf("TCP:          %s\n", color(colorRed, "Unreachable"))
		if info.Error != "" {
			fmt.Printf("Error:        %s\n", color(colorRed, info.Error))
		}
		fmt.Println()
		return nil
	}

	if info.Connected {
		fmt.Printf("Modbus:       %s\n", color(colorGreen, "Connected"))
	} else {
		fmt.Printf("Modbus:       %s\n", color(colorRed, "Failed"))
		if info.Error != "" {
			fmt.Printf("Error:        %s\n", color(colorRed, info.Error))
		}
		fmt.Println()
		return nil
	}

	fmt.Printf("Latency:      %dms\n", info.Latency.Milliseconds())

	if info.ServerID != "" {
		fmt.Printf("Server ID:    %s\n", info.ServerID)
	} else if info.ServerIDHex != "" {
		fmt.Printf("Server ID:    %s (hex)\n", info.ServerIDHex)
	}

	if info.RunStatus != "" {
		statusColor := colorGreen
		if info.RunStatus == "Stopped" {
			statusColor = colorYellow
		}
		fmt.Printf("Run Status:   %s\n", color(statusColor, info.RunStatus))
	}

	if info.Exception != 0 {
		fmt.Printf("Exception:    0x%02X (%08b)\n", info.Exception, info.Exception)
	}

	if len(info.Capabilities) > 0 {
		fmt.Println()
		fmt.Println("Supported Functions:")
		for _, cap := range info.Capabilities {
			fmt.Printf("  %s %s\n", color(colorGreen, "OK"), cap)
		}
	}

	if info.Error != "" && info.Connected {
		fmt.Println()
		fmt.Printf("Note: %s\n", color(colorYellow, info.Error))
	}

	fmt.Println()
	return nil
}
