package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	diagSubFunc uint16
	diagData    string
)

var diagCmd = &cobra.Command{
	Use:   "diag",
	Short: "Diagnostic functions",
	Long:  `Execute Modbus diagnostic functions (FC07, FC08, FC11, FC17).`,
}

var diagExceptionStatusCmd = &cobra.Command{
	Use:     "exception-status",
	Aliases: []string{"es", "exception"},
	Short:   "Read exception status (FC07)",
	RunE:    runExceptionStatus,
}

var diagDiagnosticsCmd = &cobra.Command{
	Use:     "diagnostics",
	Aliases: []string{"d"},
	Short:   "Execute diagnostics (FC08)",
	Long: `Execute diagnostics function (FC08) with various sub-functions.

Sub-functions:
  0  - Return Query Data (echo test)
  1  - Restart Communications
  10 - Clear Counters
  11 - Return Bus Message Count
  12 - Return Bus Communication Error Count
  13 - Return Bus Exception Error Count`,
	Example: `  modbuscli diag diagnostics -s 0 -d "Hello"
  modbuscli diag diagnostics -s 11`,
	RunE: runDiagnostics,
}

var diagCommEventCounterCmd = &cobra.Command{
	Use:     "comm-event-counter",
	Aliases: []string{"cec", "events"},
	Short:   "Get communication event counter (FC11)",
	RunE:    runCommEventCounter,
}

var diagServerIDCmd = &cobra.Command{
	Use:     "server-id",
	Aliases: []string{"id"},
	Short:   "Report server ID (FC17)",
	RunE:    runServerID,
}

func init() {
	diagCmd.AddCommand(diagExceptionStatusCmd)
	diagCmd.AddCommand(diagDiagnosticsCmd)
	diagCmd.AddCommand(diagCommEventCounterCmd)
	diagCmd.AddCommand(diagServerIDCmd)

	diagDiagnosticsCmd.Flags().Uint16VarP(&diagSubFunc, "subfunc", "s", 0, "Sub-function code")
	diagDiagnosticsCmd.Flags().StringVarP(&diagData, "data", "d", "", "Data to send (for echo test)")
}

func runExceptionStatus(cmd *cobra.Command, args []string) error {
	client, err := createClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	status, err := client.ReadExceptionStatus(ctx)
	if err != nil {
		return fmt.Errorf("read exception status failed: %w", err)
	}

	if outputFmt == "json" {
		data := map[string]interface{}{
			"status":     status,
			"status_hex": fmt.Sprintf("0x%02X", status),
			"bits":       decodeBits(status),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	fmt.Println()
	fmt.Println(color(colorBold, "Exception Status (FC07)"))
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Status: %d (0x%02X)\n", status, status)
	fmt.Printf("Binary: %08b\n", status)
	fmt.Println()
	fmt.Println("Bit Values:")
	for i := 0; i < 8; i++ {
		val := (status >> i) & 1
		state := color(colorRed, "OFF")
		if val == 1 {
			state = color(colorGreen, "ON")
		}
		fmt.Printf("  Bit %d: %s\n", i, state)
	}
	fmt.Println()
	return nil
}

func runDiagnostics(cmd *cobra.Command, args []string) error {
	client, err := createClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	data := []byte(diagData)
	resp, err := client.Diagnostics(ctx, diagSubFunc, data)
	if err != nil {
		return fmt.Errorf("diagnostics failed: %w", err)
	}

	subFuncName := getDiagSubFuncName(diagSubFunc)

	if outputFmt == "json" {
		result := map[string]interface{}{
			"sub_function":      diagSubFunc,
			"sub_function_name": subFuncName,
			"request_data":      fmt.Sprintf("% X", data),
			"response_data":     fmt.Sprintf("% X", resp),
		}
		if diagSubFunc == 0 {
			result["echo_match"] = string(data) == string(resp)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Println()
	fmt.Println(color(colorBold, "Diagnostics (FC08)"))
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Sub-function: %d (%s)\n", diagSubFunc, subFuncName)

	if len(data) > 0 {
		fmt.Printf("Request:      % X", data)
		if isPrintable(data) {
			fmt.Printf(" (%q)", string(data))
		}
		fmt.Println()
	}

	fmt.Printf("Response:     % X", resp)
	if isPrintable(resp) {
		fmt.Printf(" (%q)", string(resp))
	}
	fmt.Println()

	if diagSubFunc == 0 && len(data) > 0 {
		if string(data) == string(resp) {
			outputSuccess("Echo test passed")
		} else {
			outputError("Echo test failed - data mismatch")
		}
	}

	if diagSubFunc >= 11 && diagSubFunc <= 18 && len(resp) >= 2 {
		value := uint16(resp[0])<<8 | uint16(resp[1])
		fmt.Printf("Counter:      %d\n", value)
	}

	fmt.Println()
	return nil
}

func runCommEventCounter(cmd *cobra.Command, args []string) error {
	client, err := createClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	status, eventCount, err := client.GetCommEventCounter(ctx)
	if err != nil {
		return fmt.Errorf("get comm event counter failed: %w", err)
	}

	if outputFmt == "json" {
		data := map[string]interface{}{
			"status":      status,
			"event_count": eventCount,
			"busy":        status == 0xFFFF,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	fmt.Println()
	fmt.Println(color(colorBold, "Communication Event Counter (FC11)"))
	fmt.Println(strings.Repeat("-", 40))

	statusStr := color(colorGreen, "Ready")
	if status == 0xFFFF {
		statusStr = color(colorYellow, "Busy")
	}
	fmt.Printf("Status:       %s (0x%04X)\n", statusStr, status)
	fmt.Printf("Event Count:  %d\n", eventCount)
	fmt.Println()
	return nil
}

func runServerID(cmd *cobra.Command, args []string) error {
	client, err := createClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	data, err := client.ReportServerID(ctx)
	if err != nil {
		return fmt.Errorf("report server ID failed: %w", err)
	}

	if outputFmt == "json" {
		result := map[string]interface{}{
			"raw_hex": fmt.Sprintf("% X", data),
			"raw_len": len(data),
		}
		if isPrintable(data) {
			result["ascii"] = strings.TrimRight(string(data), "\x00")
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Println()
	fmt.Println(color(colorBold, "Server Identification (FC17)"))
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Length:  %d bytes\n", len(data))
	fmt.Printf("Raw:     % X\n", data)
	if isPrintable(data) {
		fmt.Printf("ASCII:   %s\n", strings.TrimRight(string(data), "\x00"))
	}

	if len(data) >= 2 {
		fmt.Println()
		fmt.Println("Decoded fields:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  Server ID:\t%d\n", data[0])
		runStatus := "OFF"
		if len(data) >= 2 && data[1] == 0xFF {
			runStatus = "ON"
		}
		fmt.Fprintf(w, "  Run Status:\t%s\n", runStatus)
		if len(data) > 2 {
			additional := data[2:]
			fmt.Fprintf(w, "  Additional:\t% X\n", additional)
			if isPrintable(additional) {
				fmt.Fprintf(w, "  (as text):\t%s\n", strings.TrimRight(string(additional), "\x00"))
			}
		}
		w.Flush()
	}
	fmt.Println()
	return nil
}

func getDiagSubFuncName(sf uint16) string {
	names := map[uint16]string{
		0:  "Return Query Data",
		1:  "Restart Communications",
		2:  "Return Diagnostic Register",
		10: "Clear Counters",
		11: "Return Bus Message Count",
		12: "Return Bus Comm Error Count",
		13: "Return Bus Exception Error Count",
		14: "Return Server Message Count",
		15: "Return Server No Response Count",
		16: "Return Server NAK Count",
		17: "Return Server Busy Count",
		18: "Return Bus Character Overrun Count",
	}
	if name, ok := names[sf]; ok {
		return name
	}
	return "Unknown"
}

func decodeBits(b uint8) []int {
	var bits []int
	for i := 0; i < 8; i++ {
		if (b>>i)&1 == 1 {
			bits = append(bits, i)
		}
	}
	return bits
}

func isPrintable(data []byte) bool {
	for _, b := range data {
		if b != 0 && (b < 32 || b > 126) {
			return false
		}
	}
	return true
}
