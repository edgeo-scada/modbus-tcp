package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	writeAddr   uint16
	writeValues []string
)

var writeCmd = &cobra.Command{
	Use:     "write",
	Aliases: []string{"w"},
	Short:   "Write data to Modbus device",
	Long:    `Write coils or registers to a Modbus device.`,
}

// Write single coil (FC05)
var writeCoilCmd = &cobra.Command{
	Use:     "coil",
	Aliases: []string{"c"},
	Short:   "Write single coil (FC05)",
	Long: `Write a single coil (discrete output) to the Modbus device using function code 05.

Value can be: 1, 0, true, false, on, off`,
	Example: `  modbuscli write coil -a 0 -V 1 -H 192.168.1.100
  modbuscli w c -a 100 -V on
  modbuscli w c -a 50 -V false`,
	RunE: runWriteCoil,
}

// Write multiple coils (FC15)
var writeCoilsCmd = &cobra.Command{
	Use:     "coils",
	Aliases: []string{"cs"},
	Short:   "Write multiple coils (FC15)",
	Long: `Write multiple coils (discrete outputs) to the Modbus device using function code 15.

Values can be comma-separated: 1,0,1,1 or 1 0 1 1`,
	Example: `  modbuscli write coils -a 0 -V 1,0,1,1,0 -H 192.168.1.100
  modbuscli w cs -a 100 -V "1 0 1 1"`,
	RunE: runWriteCoils,
}

// Write single register (FC06)
var writeRegisterCmd = &cobra.Command{
	Use:     "register",
	Aliases: []string{"reg", "r"},
	Short:   "Write single register (FC06)",
	Long: `Write a single holding register to the Modbus device using function code 06.

Value can be decimal, hexadecimal (0x prefix), or binary (0b prefix).`,
	Example: `  modbuscli write register -a 0 -V 1234 -H 192.168.1.100
  modbuscli w r -a 100 -V 0xFF00
  modbuscli w r -a 50 -V 0b1010101010101010`,
	RunE: runWriteRegister,
}

// Write multiple registers (FC16)
var writeRegistersCmd = &cobra.Command{
	Use:     "registers",
	Aliases: []string{"regs", "rs"},
	Short:   "Write multiple registers (FC16)",
	Long: `Write multiple holding registers to the Modbus device using function code 16.

Values can be comma-separated or space-separated.
Each value can be decimal, hexadecimal (0x prefix), or binary (0b prefix).`,
	Example: `  modbuscli write registers -a 0 -V 100,200,300 -H 192.168.1.100
  modbuscli w rs -a 100 -V "0x1234 0x5678"
  modbuscli w rs -a 50 -V 1000,2000,3000,4000`,
	RunE: runWriteRegisters,
}

func init() {
	// Add subcommands
	writeCmd.AddCommand(writeCoilCmd)
	writeCmd.AddCommand(writeCoilsCmd)
	writeCmd.AddCommand(writeRegisterCmd)
	writeCmd.AddCommand(writeRegistersCmd)

	// Common flags
	for _, cmd := range []*cobra.Command{writeCoilCmd, writeCoilsCmd, writeRegisterCmd, writeRegistersCmd} {
		cmd.Flags().Uint16VarP(&writeAddr, "address", "a", 0, "Starting address")
		cmd.Flags().StringSliceVarP(&writeValues, "values", "V", nil, "Values to write")
		cmd.MarkFlagRequired("values")
	}
}

func runWriteCoil(cmd *cobra.Command, args []string) error {
	if len(writeValues) == 0 {
		return fmt.Errorf("value required")
	}

	value, err := parseBoolValue(writeValues[0])
	if err != nil {
		return fmt.Errorf("invalid coil value: %w", err)
	}

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

	if err := client.WriteSingleCoil(ctx, writeAddr, value); err != nil {
		return fmt.Errorf("write coil failed: %w", err)
	}

	outputSuccess("Wrote coil %d = %v", writeAddr, value)
	return nil
}

func runWriteCoils(cmd *cobra.Command, args []string) error {
	values, err := parseBoolValues(writeValues)
	if err != nil {
		return fmt.Errorf("invalid coil values: %w", err)
	}

	if len(values) == 0 {
		return fmt.Errorf("at least one value required")
	}

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

	if err := client.WriteMultipleCoils(ctx, writeAddr, values); err != nil {
		return fmt.Errorf("write coils failed: %w", err)
	}

	outputSuccess("Wrote %d coils starting at address %d", len(values), writeAddr)
	return nil
}

func runWriteRegister(cmd *cobra.Command, args []string) error {
	if len(writeValues) == 0 {
		return fmt.Errorf("value required")
	}

	value, err := parseUint16Value(writeValues[0])
	if err != nil {
		return fmt.Errorf("invalid register value: %w", err)
	}

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

	if err := client.WriteSingleRegister(ctx, writeAddr, value); err != nil {
		return fmt.Errorf("write register failed: %w", err)
	}

	outputSuccess("Wrote register %d = %d (0x%04X)", writeAddr, value, value)
	return nil
}

func runWriteRegisters(cmd *cobra.Command, args []string) error {
	values, err := parseUint16Values(writeValues)
	if err != nil {
		return fmt.Errorf("invalid register values: %w", err)
	}

	if len(values) == 0 {
		return fmt.Errorf("at least one value required")
	}

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

	if err := client.WriteMultipleRegisters(ctx, writeAddr, values); err != nil {
		return fmt.Errorf("write registers failed: %w", err)
	}

	outputSuccess("Wrote %d registers starting at address %d", len(values), writeAddr)
	return nil
}

func parseBoolValue(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "1", "true", "on", "yes":
		return true, nil
	case "0", "false", "off", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", s)
	}
}

func parseBoolValues(values []string) ([]bool, error) {
	var result []bool
	for _, v := range values {
		// Split on comma and space
		parts := strings.FieldsFunc(v, func(r rune) bool {
			return r == ',' || r == ' '
		})
		for _, p := range parts {
			if p == "" {
				continue
			}
			b, err := parseBoolValue(p)
			if err != nil {
				return nil, err
			}
			result = append(result, b)
		}
	}
	return result, nil
}

func parseUint16Value(s string) (uint16, error) {
	s = strings.TrimSpace(s)

	var value uint64
	var err error

	switch {
	case strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X"):
		value, err = strconv.ParseUint(s[2:], 16, 16)
	case strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B"):
		value, err = strconv.ParseUint(s[2:], 2, 16)
	case strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O"):
		value, err = strconv.ParseUint(s[2:], 8, 16)
	default:
		value, err = strconv.ParseUint(s, 10, 16)
	}

	if err != nil {
		return 0, fmt.Errorf("invalid uint16 value: %s", s)
	}
	return uint16(value), nil
}

func parseUint16Values(values []string) ([]uint16, error) {
	var result []uint16
	for _, v := range values {
		// Split on comma and space
		parts := strings.FieldsFunc(v, func(r rune) bool {
			return r == ',' || r == ' '
		})
		for _, p := range parts {
			if p == "" {
				continue
			}
			u, err := parseUint16Value(p)
			if err != nil {
				return nil, err
			}
			result = append(result, u)
		}
	}
	return result, nil
}
