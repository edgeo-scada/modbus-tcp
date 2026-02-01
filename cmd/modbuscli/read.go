package main

import (
	"context"
	"fmt"

	"github.com/edgeo/drivers/modbus"
	"github.com/spf13/cobra"
)

var (
	readAddr   uint16
	readCount  uint16
	readFormat string
)

var readCmd = &cobra.Command{
	Use:     "read",
	Aliases: []string{"r"},
	Short:   "Read data from Modbus device",
	Long:    `Read coils, discrete inputs, holding registers, or input registers from a Modbus device.`,
}

// Read coils (FC01)
var readCoilsCmd = &cobra.Command{
	Use:     "coils",
	Aliases: []string{"c", "coil"},
	Short:   "Read coils (FC01)",
	Long:    `Read coils (discrete outputs) from the Modbus device using function code 01.`,
	Example: `  modbuscli read coils -a 0 -c 10 -H 192.168.1.100
  modbuscli r c -a 100 -c 8`,
	RunE: runReadCoils,
}

// Read discrete inputs (FC02)
var readDiscreteInputsCmd = &cobra.Command{
	Use:     "discrete-inputs",
	Aliases: []string{"di", "discrete"},
	Short:   "Read discrete inputs (FC02)",
	Long:    `Read discrete inputs from the Modbus device using function code 02.`,
	Example: `  modbuscli read discrete-inputs -a 0 -c 10 -H 192.168.1.100
  modbuscli r di -a 100 -c 8`,
	RunE: runReadDiscreteInputs,
}

// Read holding registers (FC03)
var readHoldingRegistersCmd = &cobra.Command{
	Use:     "holding-registers",
	Aliases: []string{"hr", "holding"},
	Short:   "Read holding registers (FC03)",
	Long: `Read holding registers from the Modbus device using function code 03.

Supported formats for -f/--format flag:
  uint16  - Unsigned 16-bit integer (default)
  int16   - Signed 16-bit integer
  uint32  - Unsigned 32-bit integer (2 registers)
  int32   - Signed 32-bit integer (2 registers)
  float32 - 32-bit floating point (2 registers)
  float64 - 64-bit floating point (4 registers)
  string  - ASCII string`,
	Example: `  modbuscli read holding-registers -a 0 -c 10 -H 192.168.1.100
  modbuscli r hr -a 100 -c 4 -f float32
  modbuscli r hr -a 0 -c 20 -f string`,
	RunE: runReadHoldingRegisters,
}

// Read input registers (FC04)
var readInputRegistersCmd = &cobra.Command{
	Use:     "input-registers",
	Aliases: []string{"ir", "input"},
	Short:   "Read input registers (FC04)",
	Long: `Read input registers from the Modbus device using function code 04.

Supported formats for -f/--format flag:
  uint16  - Unsigned 16-bit integer (default)
  int16   - Signed 16-bit integer
  uint32  - Unsigned 32-bit integer (2 registers)
  int32   - Signed 32-bit integer (2 registers)
  float32 - 32-bit floating point (2 registers)
  float64 - 64-bit floating point (4 registers)
  string  - ASCII string`,
	Example: `  modbuscli read input-registers -a 0 -c 10 -H 192.168.1.100
  modbuscli r ir -a 100 -c 4 -f int32`,
	RunE: runReadInputRegisters,
}

func init() {
	// Add subcommands
	readCmd.AddCommand(readCoilsCmd)
	readCmd.AddCommand(readDiscreteInputsCmd)
	readCmd.AddCommand(readHoldingRegistersCmd)
	readCmd.AddCommand(readInputRegistersCmd)

	// Common flags for all read commands
	for _, cmd := range []*cobra.Command{readCoilsCmd, readDiscreteInputsCmd, readHoldingRegistersCmd, readInputRegistersCmd} {
		cmd.Flags().Uint16VarP(&readAddr, "address", "a", 0, "Starting address")
		cmd.Flags().Uint16VarP(&readCount, "count", "c", 1, "Number of items to read")
	}

	// Format flag only for register commands
	readHoldingRegistersCmd.Flags().StringVarP(&readFormat, "format", "f", "uint16", "Data format: uint16, int16, uint32, int32, float32, float64, string")
	readInputRegistersCmd.Flags().StringVarP(&readFormat, "format", "f", "uint16", "Data format: uint16, int16, uint32, int32, float32, float64, string")
}

func runReadCoils(cmd *cobra.Command, args []string) error {
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

	values, err := client.ReadCoils(ctx, readAddr, readCount)
	if err != nil {
		return fmt.Errorf("read coils failed: %w", err)
	}

	return outputBoolValues("Coils", readAddr, values)
}

func runReadDiscreteInputs(cmd *cobra.Command, args []string) error {
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

	values, err := client.ReadDiscreteInputs(ctx, readAddr, readCount)
	if err != nil {
		return fmt.Errorf("read discrete inputs failed: %w", err)
	}

	return outputBoolValues("Discrete Inputs", readAddr, values)
}

func runReadHoldingRegisters(cmd *cobra.Command, args []string) error {
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

	values, err := client.ReadHoldingRegisters(ctx, readAddr, readCount)
	if err != nil {
		return fmt.Errorf("read holding registers failed: %w", err)
	}

	return outputRegisterValues("Holding Registers", readAddr, values, readFormat)
}

func runReadInputRegisters(cmd *cobra.Command, args []string) error {
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

	values, err := client.ReadInputRegisters(ctx, readAddr, readCount)
	if err != nil {
		return fmt.Errorf("read input registers failed: %w", err)
	}

	return outputRegisterValues("Input Registers", readAddr, values, readFormat)
}

func createClient() (*modbus.Client, error) {
	client, err := modbus.NewClient(
		getAddress(),
		modbus.WithUnitID(modbus.UnitID(unitID)),
		modbus.WithTimeout(timeout),
		modbus.WithAutoReconnect(retries > 1),
		modbus.WithMaxRetries(retries),
		modbus.WithLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return client, nil
}
