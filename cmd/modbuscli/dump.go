package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/edgeo/drivers/modbus"
	"github.com/spf13/cobra"
)

var (
	dumpStartAddr uint16
	dumpEndAddr   uint16
	dumpBatchSize uint16
	dumpOutFile   string
	dumpShowEmpty bool
)

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump register ranges",
	Long: `Dump a range of registers from the Modbus device.

Supports exporting to various formats including hexdump, CSV, and JSON.
Automatically handles large ranges by reading in batches.`,
	Example: `  modbuscli dump hr -a 0 -e 999 -H 192.168.1.100
  modbuscli dump ir -a 0 -e 100 -f registers.csv
  modbuscli dump coils -a 0 -e 100`,
}

var dumpHoldingCmd = &cobra.Command{
	Use:     "holding-registers",
	Aliases: []string{"hr", "holding"},
	Short:   "Dump holding registers",
	RunE:    runDumpHoldingRegisters,
}

var dumpInputCmd = &cobra.Command{
	Use:     "input-registers",
	Aliases: []string{"ir", "input"},
	Short:   "Dump input registers",
	RunE:    runDumpInputRegisters,
}

var dumpCoilsCmd = &cobra.Command{
	Use:     "coils",
	Aliases: []string{"c", "coil"},
	Short:   "Dump coils",
	RunE:    runDumpCoils,
}

var dumpDiscreteCmd = &cobra.Command{
	Use:     "discrete-inputs",
	Aliases: []string{"di", "discrete"},
	Short:   "Dump discrete inputs",
	RunE:    runDumpDiscreteInputs,
}

func init() {
	dumpCmd.AddCommand(dumpHoldingCmd)
	dumpCmd.AddCommand(dumpInputCmd)
	dumpCmd.AddCommand(dumpCoilsCmd)
	dumpCmd.AddCommand(dumpDiscreteCmd)

	for _, cmd := range []*cobra.Command{dumpHoldingCmd, dumpInputCmd} {
		cmd.Flags().Uint16VarP(&dumpStartAddr, "start", "a", 0, "Start address")
		cmd.Flags().Uint16VarP(&dumpEndAddr, "end", "e", 100, "End address")
		cmd.Flags().Uint16VarP(&dumpBatchSize, "batch", "b", 125, "Batch size for reading")
		cmd.Flags().StringVarP(&dumpOutFile, "file", "f", "", "Output file (default: stdout)")
		cmd.Flags().BoolVar(&dumpShowEmpty, "show-empty", false, "Show addresses that return errors")
	}

	for _, cmd := range []*cobra.Command{dumpCoilsCmd, dumpDiscreteCmd} {
		cmd.Flags().Uint16VarP(&dumpStartAddr, "start", "a", 0, "Start address")
		cmd.Flags().Uint16VarP(&dumpEndAddr, "end", "e", 100, "End address")
		cmd.Flags().Uint16VarP(&dumpBatchSize, "batch", "b", 2000, "Batch size for reading")
		cmd.Flags().StringVarP(&dumpOutFile, "file", "f", "", "Output file (default: stdout)")
		cmd.Flags().BoolVar(&dumpShowEmpty, "show-empty", false, "Show addresses that return errors")
	}
}

func runDumpHoldingRegisters(cmd *cobra.Command, args []string) error {
	return dumpRegisters(func(ctx context.Context, client *modbus.Client, addr, qty uint16) ([]uint16, error) {
		return client.ReadHoldingRegisters(ctx, addr, qty)
	}, "Holding Registers")
}

func runDumpInputRegisters(cmd *cobra.Command, args []string) error {
	return dumpRegisters(func(ctx context.Context, client *modbus.Client, addr, qty uint16) ([]uint16, error) {
		return client.ReadInputRegisters(ctx, addr, qty)
	}, "Input Registers")
}

func runDumpCoils(cmd *cobra.Command, args []string) error {
	return dumpBools(func(ctx context.Context, client *modbus.Client, addr, qty uint16) ([]bool, error) {
		return client.ReadCoils(ctx, addr, qty)
	}, "Coils")
}

func runDumpDiscreteInputs(cmd *cobra.Command, args []string) error {
	return dumpBools(func(ctx context.Context, client *modbus.Client, addr, qty uint16) ([]bool, error) {
		return client.ReadDiscreteInputs(ctx, addr, qty)
	}, "Discrete Inputs")
}

type DumpRegister struct {
	Address uint16 `json:"address"`
	Value   uint16 `json:"value"`
	Hex     string `json:"hex"`
	Error   string `json:"error,omitempty"`
}

type DumpBool struct {
	Address uint16 `json:"address"`
	Value   bool   `json:"value"`
	Error   string `json:"error,omitempty"`
}

func dumpRegisters(readFunc func(context.Context, *modbus.Client, uint16, uint16) ([]uint16, error), title string) error {
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

	if dumpEndAddr < dumpStartAddr {
		dumpStartAddr, dumpEndAddr = dumpEndAddr, dumpStartAddr
	}

	totalCount := int(dumpEndAddr - dumpStartAddr + 1)
	results := make([]DumpRegister, 0, totalCount)

	outputInfo("Dumping %s from %d to %d (%d registers)...", title, dumpStartAddr, dumpEndAddr, totalCount)
	startTime := time.Now()

	for addr := dumpStartAddr; addr <= dumpEndAddr; {
		batchSize := dumpBatchSize
		if addr+batchSize > dumpEndAddr+1 {
			batchSize = dumpEndAddr - addr + 1
		}

		readCtx, readCancel := context.WithTimeout(ctx, timeout)
		values, err := readFunc(readCtx, client, addr, batchSize)
		readCancel()

		if err != nil {
			if dumpShowEmpty {
				for i := uint16(0); i < batchSize; i++ {
					results = append(results, DumpRegister{
						Address: addr + i,
						Error:   err.Error(),
					})
				}
			}
		} else {
			for i, v := range values {
				results = append(results, DumpRegister{
					Address: addr + uint16(i),
					Value:   v,
					Hex:     fmt.Sprintf("0x%04X", v),
				})
			}
		}

		addr += batchSize

		if verbose {
			progress := float64(addr-dumpStartAddr) / float64(totalCount) * 100
			fmt.Fprintf(os.Stderr, "\rProgress: %.1f%%", progress)
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr)
	}

	duration := time.Since(startTime)
	outputInfo("Read %d registers in %s (%.1f regs/sec)", len(results), duration.Round(time.Millisecond), float64(len(results))/duration.Seconds())

	return outputDumpRegisters(title, results)
}

func dumpBools(readFunc func(context.Context, *modbus.Client, uint16, uint16) ([]bool, error), title string) error {
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

	if dumpEndAddr < dumpStartAddr {
		dumpStartAddr, dumpEndAddr = dumpEndAddr, dumpStartAddr
	}

	totalCount := int(dumpEndAddr - dumpStartAddr + 1)
	results := make([]DumpBool, 0, totalCount)

	outputInfo("Dumping %s from %d to %d (%d items)...", title, dumpStartAddr, dumpEndAddr, totalCount)
	startTime := time.Now()

	batchSize := dumpBatchSize
	if batchSize > 2000 {
		batchSize = 2000
	}

	for addr := dumpStartAddr; addr <= dumpEndAddr; {
		bs := batchSize
		if addr+bs > dumpEndAddr+1 {
			bs = dumpEndAddr - addr + 1
		}

		readCtx, readCancel := context.WithTimeout(ctx, timeout)
		values, err := readFunc(readCtx, client, addr, bs)
		readCancel()

		if err != nil {
			if dumpShowEmpty {
				for i := uint16(0); i < bs; i++ {
					results = append(results, DumpBool{
						Address: addr + i,
						Error:   err.Error(),
					})
				}
			}
		} else {
			for i, v := range values {
				results = append(results, DumpBool{
					Address: addr + uint16(i),
					Value:   v,
				})
			}
		}

		addr += bs
	}

	duration := time.Since(startTime)
	outputInfo("Read %d items in %s", len(results), duration.Round(time.Millisecond))

	return outputDumpBools(title, results)
}

func outputDumpRegisters(title string, results []DumpRegister) error {
	var out *os.File = os.Stdout
	if dumpOutFile != "" {
		f, err := os.Create(dumpOutFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	switch outputFmt {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(results)

	case "csv":
		w := csv.NewWriter(out)
		w.Write([]string{"address", "value", "hex", "error"})
		for _, r := range results {
			w.Write([]string{
				fmt.Sprintf("%d", r.Address),
				fmt.Sprintf("%d", r.Value),
				r.Hex,
				r.Error,
			})
		}
		w.Flush()
		return w.Error()

	case "hex":
		for i := 0; i < len(results); i += 8 {
			end := i + 8
			if end > len(results) {
				end = len(results)
			}
			fmt.Fprintf(out, "%08x  ", results[i].Address)
			for j := i; j < end; j++ {
				if results[j].Error != "" {
					fmt.Fprint(out, "?? ?? ")
				} else {
					fmt.Fprintf(out, "%02x %02x ", results[j].Value>>8, results[j].Value&0xFF)
				}
			}
			for j := end - i; j < 8; j++ {
				fmt.Fprint(out, "      ")
			}
			fmt.Fprint(out, " |")
			for j := i; j < end; j++ {
				if results[j].Error != "" {
					fmt.Fprint(out, "..")
				} else {
					for _, b := range []byte{byte(results[j].Value >> 8), byte(results[j].Value & 0xFF)} {
						if b >= 32 && b < 127 {
							fmt.Fprintf(out, "%c", b)
						} else {
							fmt.Fprint(out, ".")
						}
					}
				}
			}
			fmt.Fprintln(out, "|")
		}
		return nil

	default:
		fmt.Fprintf(out, "\n%s Dump\n", title)
		fmt.Fprintln(out, strings.Repeat("=", 60))

		for i := 0; i < len(results); i += 16 {
			end := i + 16
			if end > len(results) {
				end = len(results)
			}
			fmt.Fprintf(out, "%5d: ", results[i].Address)
			for j := i; j < end; j++ {
				if results[j].Error != "" {
					fmt.Fprintf(out, " ---- ")
				} else {
					fmt.Fprintf(out, " %04X ", results[j].Value)
				}
			}
			for j := end - i; j < 16; j++ {
				fmt.Fprintf(out, "      ")
			}
			fmt.Fprint(out, " |")
			for j := i; j < end; j++ {
				if results[j].Error != "" {
					fmt.Fprint(out, "..")
				} else {
					hi := byte(results[j].Value >> 8)
					lo := byte(results[j].Value & 0xFF)
					if hi >= 32 && hi < 127 {
						fmt.Fprintf(out, "%c", hi)
					} else {
						fmt.Fprint(out, ".")
					}
					if lo >= 32 && lo < 127 {
						fmt.Fprintf(out, "%c", lo)
					} else {
						fmt.Fprint(out, ".")
					}
				}
			}
			fmt.Fprintln(out, "|")
		}
		fmt.Fprintln(out)
	}

	if dumpOutFile != "" {
		outputSuccess("Output written to %s", dumpOutFile)
	}
	return nil
}

func outputDumpBools(title string, results []DumpBool) error {
	var out *os.File = os.Stdout
	if dumpOutFile != "" {
		f, err := os.Create(dumpOutFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	switch outputFmt {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(results)

	case "csv":
		w := csv.NewWriter(out)
		w.Write([]string{"address", "value", "error"})
		for _, r := range results {
			val := "0"
			if r.Value {
				val = "1"
			}
			w.Write([]string{
				fmt.Sprintf("%d", r.Address),
				val,
				r.Error,
			})
		}
		w.Flush()
		return w.Error()

	default:
		fmt.Fprintf(out, "\n%s Dump\n", title)
		fmt.Fprintln(out, strings.Repeat("=", 60))

		for i := 0; i < len(results); i += 32 {
			end := i + 32
			if end > len(results) {
				end = len(results)
			}
			fmt.Fprintf(out, "%5d: ", results[i].Address)
			for j := i; j < end; j++ {
				if results[j].Error != "" {
					fmt.Fprint(out, "?")
				} else if results[j].Value {
					fmt.Fprint(out, "1")
				} else {
					fmt.Fprint(out, "0")
				}
				if (j-i+1)%8 == 0 {
					fmt.Fprint(out, " ")
				}
			}
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out)
	}

	if dumpOutFile != "" {
		outputSuccess("Output written to %s", dumpOutFile)
	}
	return nil
}
