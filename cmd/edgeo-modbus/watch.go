// Copyright 2025 Edgeo SCADA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/edgeo-scada/modbus"
	"github.com/spf13/cobra"
)

var (
	watchInterval    time.Duration
	watchCount       int
	watchShowDiff    bool
	watchClearTerm   bool
	watchTimestamp   bool
	watchLogFile     string
	watchAlertHigh   float64
	watchAlertLow    float64
	watchAlertEnable bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Continuously monitor Modbus values",
	Long: `Watch Modbus registers or coils continuously with configurable interval.

Supports:
  - Holding registers (hr)
  - Input registers (ir)
  - Coils (c)
  - Discrete inputs (di)

Features:
  - Change detection and highlighting
  - Alert thresholds
  - Logging to file
  - Timestamp display`,
	Example: `  # Watch 5 holding registers every second
  modbuscli watch hr -a 0 -c 5 -i 1s -H 192.168.1.100

  # Watch with alerts when value exceeds threshold
  modbuscli watch hr -a 100 -c 1 -i 500ms --alert-high 1000

  # Watch and log to file
  modbuscli watch hr -a 0 -c 10 -i 2s --log data.csv

  # Watch coils with change highlighting
  modbuscli watch c -a 0 -c 8 -i 1s --diff`,
}

var watchHoldingRegistersCmd = &cobra.Command{
	Use:     "holding-registers",
	Aliases: []string{"hr", "holding"},
	Short:   "Watch holding registers",
	RunE:    runWatchHoldingRegisters,
}

var watchInputRegistersCmd = &cobra.Command{
	Use:     "input-registers",
	Aliases: []string{"ir", "input"},
	Short:   "Watch input registers",
	RunE:    runWatchInputRegisters,
}

var watchCoilsCmd = &cobra.Command{
	Use:     "coils",
	Aliases: []string{"c", "coil"},
	Short:   "Watch coils",
	RunE:    runWatchCoils,
}

var watchDiscreteInputsCmd = &cobra.Command{
	Use:     "discrete-inputs",
	Aliases: []string{"di", "discrete"},
	Short:   "Watch discrete inputs",
	RunE:    runWatchDiscreteInputs,
}

func init() {
	watchCmd.AddCommand(watchHoldingRegistersCmd)
	watchCmd.AddCommand(watchInputRegistersCmd)
	watchCmd.AddCommand(watchCoilsCmd)
	watchCmd.AddCommand(watchDiscreteInputsCmd)

	for _, cmd := range []*cobra.Command{watchHoldingRegistersCmd, watchInputRegistersCmd, watchCoilsCmd, watchDiscreteInputsCmd} {
		cmd.Flags().Uint16VarP(&readAddr, "address", "a", 0, "Starting address")
		cmd.Flags().Uint16VarP(&readCount, "count", "c", 1, "Number of items to read")
		cmd.Flags().DurationVarP(&watchInterval, "interval", "i", 1*time.Second, "Poll interval")
		cmd.Flags().IntVarP(&watchCount, "iterations", "n", 0, "Number of iterations (0 = infinite)")
		cmd.Flags().BoolVar(&watchShowDiff, "diff", false, "Highlight changed values")
		cmd.Flags().BoolVar(&watchClearTerm, "clear", true, "Clear terminal between updates")
		cmd.Flags().BoolVar(&watchTimestamp, "timestamp", true, "Show timestamps")
		cmd.Flags().StringVar(&watchLogFile, "log", "", "Log values to file (CSV format)")
	}

	for _, cmd := range []*cobra.Command{watchHoldingRegistersCmd, watchInputRegistersCmd} {
		cmd.Flags().StringVarP(&readFormat, "format", "f", "uint16", "Data format")
		cmd.Flags().Float64Var(&watchAlertHigh, "alert-high", 0, "Alert when value exceeds this threshold")
		cmd.Flags().Float64Var(&watchAlertLow, "alert-low", 0, "Alert when value falls below this threshold")
		cmd.Flags().BoolVar(&watchAlertEnable, "alert", false, "Enable threshold alerts")
	}
}

type WatchState struct {
	client       *modbus.Client
	ctx          context.Context
	cancel       context.CancelFunc
	prevRegs     []uint16
	prevCoils    []bool
	iteration    int
	logFile      *os.File
	startTime    time.Time
	errorCount   int
	successCount int
}

func runWatchHoldingRegisters(cmd *cobra.Command, args []string) error {
	return watchRegisters(func(ctx context.Context, client *modbus.Client) ([]uint16, error) {
		return client.ReadHoldingRegisters(ctx, readAddr, readCount)
	}, "Holding Registers")
}

func runWatchInputRegisters(cmd *cobra.Command, args []string) error {
	return watchRegisters(func(ctx context.Context, client *modbus.Client) ([]uint16, error) {
		return client.ReadInputRegisters(ctx, readAddr, readCount)
	}, "Input Registers")
}

func runWatchCoils(cmd *cobra.Command, args []string) error {
	return watchBools(func(ctx context.Context, client *modbus.Client) ([]bool, error) {
		return client.ReadCoils(ctx, readAddr, readCount)
	}, "Coils")
}

func runWatchDiscreteInputs(cmd *cobra.Command, args []string) error {
	return watchBools(func(ctx context.Context, client *modbus.Client) ([]bool, error) {
		return client.ReadDiscreteInputs(ctx, readAddr, readCount)
	}, "Discrete Inputs")
}

func watchRegisters(readFunc func(context.Context, *modbus.Client) ([]uint16, error), title string) error {
	state, err := initWatchState()
	if err != nil {
		return err
	}
	defer state.cleanup()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	if err := state.readAndDisplayRegisters(readFunc, title); err != nil {
		outputWarning("Initial read failed: %v", err)
	}

	for {
		select {
		case <-sigCh:
			fmt.Println("\n\nStopping watch...")
			state.printSummary()
			return nil
		case <-ticker.C:
			if err := state.readAndDisplayRegisters(readFunc, title); err != nil {
				state.errorCount++
				if verbose {
					outputWarning("Read failed: %v", err)
				}
			}
			if watchCount > 0 && state.iteration >= watchCount {
				state.printSummary()
				return nil
			}
		case <-state.ctx.Done():
			return state.ctx.Err()
		}
	}
}

func watchBools(readFunc func(context.Context, *modbus.Client) ([]bool, error), title string) error {
	state, err := initWatchState()
	if err != nil {
		return err
	}
	defer state.cleanup()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	if err := state.readAndDisplayBools(readFunc, title); err != nil {
		outputWarning("Initial read failed: %v", err)
	}

	for {
		select {
		case <-sigCh:
			fmt.Println("\n\nStopping watch...")
			state.printSummary()
			return nil
		case <-ticker.C:
			if err := state.readAndDisplayBools(readFunc, title); err != nil {
				state.errorCount++
			}
			if watchCount > 0 && state.iteration >= watchCount {
				state.printSummary()
				return nil
			}
		case <-state.ctx.Done():
			return state.ctx.Err()
		}
	}
}

func initWatchState() (*WatchState, error) {
	client, err := createClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := client.Connect(ctx); err != nil {
		cancel()
		client.Close()
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	state := &WatchState{
		client:    client,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}

	if watchLogFile != "" {
		f, err := os.Create(watchLogFile)
		if err != nil {
			state.cleanup()
			return nil, fmt.Errorf("failed to create log file: %w", err)
		}
		state.logFile = f
	}

	return state, nil
}

func (s *WatchState) cleanup() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.client != nil {
		s.client.Close()
	}
	if s.logFile != nil {
		s.logFile.Close()
	}
}

func (s *WatchState) readAndDisplayRegisters(readFunc func(context.Context, *modbus.Client) ([]uint16, error), title string) error {
	readCtx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	values, err := readFunc(readCtx, s.client)
	if err != nil {
		return err
	}

	s.iteration++
	s.successCount++

	now := time.Now()

	if outputFmt == "json" {
		return s.outputWatchJSON(values, now)
	}

	if watchClearTerm && s.iteration > 1 {
		fmt.Print("\033[H\033[2J")
	}

	fmt.Printf("%s - Watching %s (Address %d-%d)\n",
		color(colorBold, "MODBUS WATCH"),
		title,
		readAddr,
		readAddr+readCount-1)
	fmt.Printf("Host: %s | Unit: %d | Interval: %s\n", getAddress(), unitID, watchInterval)
	if watchTimestamp {
		fmt.Printf("Time: %s | Iteration: %d", now.Format("15:04:05.000"), s.iteration)
		if watchCount > 0 {
			fmt.Printf("/%d", watchCount)
		}
		fmt.Println()
	}
	fmt.Println(strings.Repeat("-", 60))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ADDR\tVALUE\tHEX\tCHANGE")
	fmt.Fprintln(w, "----\t-----\t---\t------")

	for i, v := range values {
		addr := readAddr + uint16(i)
		change := ""

		if watchShowDiff && s.prevRegs != nil && i < len(s.prevRegs) {
			diff := int(v) - int(s.prevRegs[i])
			if diff > 0 {
				change = color(colorGreen, fmt.Sprintf("+%d", diff))
			} else if diff < 0 {
				change = color(colorRed, fmt.Sprintf("%d", diff))
			}
		}

		if watchAlertEnable {
			fv := float64(v)
			if watchAlertHigh != 0 && fv > watchAlertHigh {
				change += " " + color(colorRed+colorBold, "HIGH!")
			}
			if watchAlertLow != 0 && fv < watchAlertLow {
				change += " " + color(colorYellow+colorBold, "LOW!")
			}
		}

		fmt.Fprintf(w, "%d\t%d\t0x%04X\t%s\n", addr, v, v, change)
	}
	w.Flush()

	if s.logFile != nil {
		s.logToFile(now, values)
	}

	s.prevRegs = values
	return nil
}

func (s *WatchState) readAndDisplayBools(readFunc func(context.Context, *modbus.Client) ([]bool, error), title string) error {
	readCtx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	values, err := readFunc(readCtx, s.client)
	if err != nil {
		return err
	}

	s.iteration++
	s.successCount++

	now := time.Now()

	if outputFmt == "json" {
		return s.outputWatchBoolJSON(values, now)
	}

	if watchClearTerm && s.iteration > 1 {
		fmt.Print("\033[H\033[2J")
	}

	fmt.Printf("%s - Watching %s (Address %d-%d)\n",
		color(colorBold, "MODBUS WATCH"),
		title,
		readAddr,
		readAddr+readCount-1)
	fmt.Printf("Host: %s | Unit: %d | Interval: %s\n", getAddress(), unitID, watchInterval)
	if watchTimestamp {
		fmt.Printf("Time: %s | Iteration: %d", now.Format("15:04:05.000"), s.iteration)
		if watchCount > 0 {
			fmt.Printf("/%d", watchCount)
		}
		fmt.Println()
	}
	fmt.Println(strings.Repeat("-", 50))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ADDR\tVALUE\tSTATUS\tCHANGE")
	fmt.Fprintln(w, "----\t-----\t------\t------")

	for i, v := range values {
		addr := readAddr + uint16(i)
		valStr := "0"
		status := color(colorRed, "OFF")
		if v {
			valStr = "1"
			status = color(colorGreen, "ON")
		}

		change := ""
		if watchShowDiff && s.prevCoils != nil && i < len(s.prevCoils) {
			if v != s.prevCoils[i] {
				if v {
					change = color(colorGreen, "->ON")
				} else {
					change = color(colorRed, "->OFF")
				}
			}
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", addr, valStr, status, change)
	}
	w.Flush()

	s.prevCoils = values
	return nil
}

func (s *WatchState) outputWatchJSON(values []uint16, ts time.Time) error {
	data := struct {
		Timestamp string   `json:"timestamp"`
		Iteration int      `json:"iteration"`
		Address   uint16   `json:"start_address"`
		Values    []uint16 `json:"values"`
		ValuesHex []string `json:"values_hex"`
	}{
		Timestamp: ts.Format(time.RFC3339Nano),
		Iteration: s.iteration,
		Address:   readAddr,
		Values:    values,
		ValuesHex: make([]string, len(values)),
	}
	for i, v := range values {
		data.ValuesHex[i] = fmt.Sprintf("0x%04X", v)
	}
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(data)
}

func (s *WatchState) outputWatchBoolJSON(values []bool, ts time.Time) error {
	data := struct {
		Timestamp string `json:"timestamp"`
		Iteration int    `json:"iteration"`
		Address   uint16 `json:"start_address"`
		Values    []bool `json:"values"`
	}{
		Timestamp: ts.Format(time.RFC3339Nano),
		Iteration: s.iteration,
		Address:   readAddr,
		Values:    values,
	}
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(data)
}

func (s *WatchState) logToFile(ts time.Time, values []uint16) {
	if s.iteration == 1 {
		header := "timestamp"
		for i := uint16(0); i < readCount; i++ {
			header += fmt.Sprintf(",addr_%d", readAddr+i)
		}
		fmt.Fprintln(s.logFile, header)
	}

	line := ts.Format(time.RFC3339)
	for _, v := range values {
		line += fmt.Sprintf(",%d", v)
	}
	fmt.Fprintln(s.logFile, line)
}

func (s *WatchState) printSummary() {
	duration := time.Since(s.startTime)
	fmt.Println()
	fmt.Println(color(colorBold, "Watch Summary"))
	fmt.Println(strings.Repeat("-", 30))
	fmt.Printf("Duration:    %s\n", duration.Round(time.Millisecond))
	fmt.Printf("Iterations:  %d\n", s.iteration)
	fmt.Printf("Success:     %d\n", s.successCount)
	fmt.Printf("Errors:      %d\n", s.errorCount)
	if s.iteration > 0 {
		fmt.Printf("Avg Rate:    %.2f reads/sec\n", float64(s.iteration)/duration.Seconds())
	}
}
