package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/edgeo/drivers/modbus"
	"github.com/spf13/cobra"
)

var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"i", "repl", "shell"},
	Short:   "Start interactive Modbus shell",
	Long: `Start an interactive shell for Modbus communication.

Available commands:
  connect [host:port]           - Connect to server
  disconnect                    - Disconnect from server
  unit <id>                     - Set unit ID
  status                        - Show connection status

  rc <addr> [count]             - Read coils
  rdi <addr> [count]            - Read discrete inputs
  rhr <addr> [count] [format]   - Read holding registers
  rir <addr> [count] [format]   - Read input registers

  wc <addr> <value>             - Write single coil
  wr <addr> <value>             - Write single register
  wcs <addr> <v1,v2,...>        - Write multiple coils
  wrs <addr> <v1,v2,...>        - Write multiple registers

  scan [start] [end]            - Scan unit IDs
  dump <type> <start> <end>     - Dump address range

  output <format>               - Set output format (table/json/csv/hex)
  format <type>                 - Set register format (uint16/int16/etc)

  help                          - Show help
  quit                          - Exit`,
	Example: `  modbuscli interactive -H 192.168.1.100
  modbuscli i --host 10.0.0.50 --port 5020`,
	RunE: runInteractive,
}

type InteractiveSession struct {
	client      *modbus.Client
	connected   bool
	currentHost string
	currentUnit uint8
	regFormat   string
}

func runInteractive(cmd *cobra.Command, args []string) error {
	session := &InteractiveSession{
		currentHost: getAddress(),
		currentUnit: unitID,
		regFormat:   "uint16",
	}

	fmt.Println(color(colorBold, "Modbus Interactive Shell"))
	fmt.Println("Type 'help' for available commands, 'quit' to exit")
	fmt.Println()

	if host != "localhost" || port != 502 {
		if err := session.connect(session.currentHost); err != nil {
			outputWarning("Auto-connect failed: %v", err)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		prompt := session.getPrompt()
		fmt.Print(prompt)

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if err := session.execute(line); err != nil {
			if err.Error() == "quit" {
				break
			}
			outputError("%v", err)
		}
	}

	if session.client != nil {
		session.client.Close()
	}

	fmt.Println("\nGoodbye!")
	return nil
}

func (s *InteractiveSession) getPrompt() string {
	status := color(colorRed, "disconnected")
	if s.connected {
		status = color(colorGreen, s.currentHost)
	}
	return fmt.Sprintf("modbus[%s]@%d> ", status, s.currentUnit)
}

func (s *InteractiveSession) execute(line string) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "quit", "exit", "q":
		return fmt.Errorf("quit")
	case "help", "h", "?":
		s.showHelp()
		return nil
	case "connect", "conn", "c":
		addr := s.currentHost
		if len(args) > 0 {
			addr = args[0]
			if !strings.Contains(addr, ":") {
				addr = addr + ":502"
			}
		}
		return s.connect(addr)
	case "disconnect", "disc", "d":
		return s.disconnect()
	case "status", "stat", "s":
		s.showStatus()
		return nil
	case "unit", "u":
		if len(args) < 1 {
			fmt.Printf("Current unit ID: %d\n", s.currentUnit)
			return nil
		}
		id, err := strconv.Atoi(args[0])
		if err != nil || id < 1 || id > 247 {
			return fmt.Errorf("invalid unit ID (1-247)")
		}
		s.currentUnit = uint8(id)
		if s.client != nil {
			s.client.SetUnitID(modbus.UnitID(s.currentUnit))
		}
		fmt.Printf("Unit ID set to %d\n", s.currentUnit)
		return nil
	case "output", "out", "o":
		if len(args) < 1 {
			fmt.Printf("Current output format: %s\n", outputFmt)
			return nil
		}
		switch args[0] {
		case "table", "json", "csv", "hex", "raw":
			outputFmt = args[0]
			fmt.Printf("Output format set to %s\n", outputFmt)
		default:
			return fmt.Errorf("invalid format: %s", args[0])
		}
		return nil
	case "format", "fmt", "f":
		if len(args) < 1 {
			fmt.Printf("Current register format: %s\n", s.regFormat)
			return nil
		}
		s.regFormat = args[0]
		fmt.Printf("Register format set to %s\n", s.regFormat)
		return nil
	case "rc", "readcoils":
		return s.readCoils(args)
	case "rdi", "readdiscrete":
		return s.readDiscreteInputs(args)
	case "rhr", "readholding":
		return s.readHoldingRegisters(args)
	case "rir", "readinput":
		return s.readInputRegisters(args)
	case "wc", "writecoil":
		return s.writeSingleCoil(args)
	case "wr", "writereg":
		return s.writeSingleRegister(args)
	case "wcs", "writecoils":
		return s.writeMultipleCoils(args)
	case "wrs", "writeregs":
		return s.writeMultipleRegisters(args)
	case "scan":
		return s.scanUnits(args)
	case "id", "serverid":
		return s.getServerID()
	default:
		return fmt.Errorf("unknown command: %s (type 'help' for commands)", cmd)
	}
}

func (s *InteractiveSession) connect(addr string) error {
	if s.client != nil {
		s.client.Close()
	}

	client, err := modbus.NewClient(
		addr,
		modbus.WithUnitID(modbus.UnitID(s.currentUnit)),
		modbus.WithTimeout(timeout),
		modbus.WithLogger(logger),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		client.Close()
		return fmt.Errorf("connection failed: %w", err)
	}

	s.client = client
	s.currentHost = addr
	s.connected = true
	outputSuccess("Connected to %s", addr)
	return nil
}

func (s *InteractiveSession) disconnect() error {
	if s.client != nil {
		s.client.Close()
		s.client = nil
	}
	s.connected = false
	outputInfo("Disconnected")
	return nil
}

func (s *InteractiveSession) showStatus() {
	fmt.Println()
	fmt.Println(color(colorBold, "Connection Status"))
	fmt.Println(strings.Repeat("-", 30))
	if s.connected {
		fmt.Printf("Status:        %s\n", color(colorGreen, "Connected"))
		fmt.Printf("Host:          %s\n", s.currentHost)
	} else {
		fmt.Printf("Status:        %s\n", color(colorRed, "Disconnected"))
	}
	fmt.Printf("Unit ID:       %d\n", s.currentUnit)
	fmt.Printf("Output:        %s\n", outputFmt)
	fmt.Printf("Reg Format:    %s\n", s.regFormat)
	fmt.Printf("Timeout:       %s\n", timeout)
	fmt.Println()
}

func (s *InteractiveSession) showHelp() {
	help := `
Commands:
  Connection:
    connect [host:port]    Connect to Modbus server
    disconnect             Disconnect from server
    unit <id>              Set/show unit ID (1-247)
    status                 Show connection status

  Read Operations:
    rc <addr> [count]               Read coils (FC01)
    rdi <addr> [count]              Read discrete inputs (FC02)
    rhr <addr> [count] [format]     Read holding registers (FC03)
    rir <addr> [count] [format]     Read input registers (FC04)

  Write Operations:
    wc <addr> <0|1>                 Write single coil (FC05)
    wr <addr> <value>               Write single register (FC06)
    wcs <addr> <v1,v2,...>          Write multiple coils (FC15)
    wrs <addr> <v1,v2,...>          Write multiple registers (FC16)

  Tools:
    scan [start] [end]     Scan for active unit IDs
    id                     Get server identification

  Settings:
    output <format>        Set output format (table/json/csv/hex/raw)
    format <type>          Set register format (uint16/int16/uint32/int32/float32)

  General:
    help                   Show this help
    quit                   Exit interactive mode
`
	fmt.Println(help)
}

func (s *InteractiveSession) requireConnection() error {
	if !s.connected || s.client == nil {
		return fmt.Errorf("not connected (use 'connect' first)")
	}
	return nil
}

func (s *InteractiveSession) readCoils(args []string) error {
	if err := s.requireConnection(); err != nil {
		return err
	}
	addr, count := uint16(0), uint16(1)
	if len(args) >= 1 {
		a, _ := strconv.Atoi(args[0])
		addr = uint16(a)
	}
	if len(args) >= 2 {
		c, _ := strconv.Atoi(args[1])
		count = uint16(c)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	values, err := s.client.ReadCoils(ctx, addr, count)
	if err != nil {
		return err
	}
	return outputBoolValues("Coils", addr, values)
}

func (s *InteractiveSession) readDiscreteInputs(args []string) error {
	if err := s.requireConnection(); err != nil {
		return err
	}
	addr, count := uint16(0), uint16(1)
	if len(args) >= 1 {
		a, _ := strconv.Atoi(args[0])
		addr = uint16(a)
	}
	if len(args) >= 2 {
		c, _ := strconv.Atoi(args[1])
		count = uint16(c)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	values, err := s.client.ReadDiscreteInputs(ctx, addr, count)
	if err != nil {
		return err
	}
	return outputBoolValues("Discrete Inputs", addr, values)
}

func (s *InteractiveSession) readHoldingRegisters(args []string) error {
	if err := s.requireConnection(); err != nil {
		return err
	}
	addr, count := uint16(0), uint16(1)
	format := s.regFormat
	if len(args) >= 1 {
		a, _ := strconv.Atoi(args[0])
		addr = uint16(a)
	}
	if len(args) >= 2 {
		c, _ := strconv.Atoi(args[1])
		count = uint16(c)
	}
	if len(args) >= 3 {
		format = args[2]
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	values, err := s.client.ReadHoldingRegisters(ctx, addr, count)
	if err != nil {
		return err
	}
	return outputRegisterValues("Holding Registers", addr, values, format)
}

func (s *InteractiveSession) readInputRegisters(args []string) error {
	if err := s.requireConnection(); err != nil {
		return err
	}
	addr, count := uint16(0), uint16(1)
	format := s.regFormat
	if len(args) >= 1 {
		a, _ := strconv.Atoi(args[0])
		addr = uint16(a)
	}
	if len(args) >= 2 {
		c, _ := strconv.Atoi(args[1])
		count = uint16(c)
	}
	if len(args) >= 3 {
		format = args[2]
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	values, err := s.client.ReadInputRegisters(ctx, addr, count)
	if err != nil {
		return err
	}
	return outputRegisterValues("Input Registers", addr, values, format)
}

func (s *InteractiveSession) writeSingleCoil(args []string) error {
	if err := s.requireConnection(); err != nil {
		return err
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: wc <address> <0|1>")
	}
	addr, _ := strconv.Atoi(args[0])
	value, err := parseBoolValue(args[1])
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.client.WriteSingleCoil(ctx, uint16(addr), value); err != nil {
		return err
	}
	outputSuccess("Wrote coil %d = %v", addr, value)
	return nil
}

func (s *InteractiveSession) writeSingleRegister(args []string) error {
	if err := s.requireConnection(); err != nil {
		return err
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: wr <address> <value>")
	}
	addr, _ := strconv.Atoi(args[0])
	value, err := parseUint16Value(args[1])
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.client.WriteSingleRegister(ctx, uint16(addr), value); err != nil {
		return err
	}
	outputSuccess("Wrote register %d = %d (0x%04X)", addr, value, value)
	return nil
}

func (s *InteractiveSession) writeMultipleCoils(args []string) error {
	if err := s.requireConnection(); err != nil {
		return err
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: wcs <address> <v1,v2,...>")
	}
	addr, _ := strconv.Atoi(args[0])
	values, err := parseBoolValues(args[1:])
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.client.WriteMultipleCoils(ctx, uint16(addr), values); err != nil {
		return err
	}
	outputSuccess("Wrote %d coils starting at address %d", len(values), addr)
	return nil
}

func (s *InteractiveSession) writeMultipleRegisters(args []string) error {
	if err := s.requireConnection(); err != nil {
		return err
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: wrs <address> <v1,v2,...>")
	}
	addr, _ := strconv.Atoi(args[0])
	values, err := parseUint16Values(args[1:])
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.client.WriteMultipleRegisters(ctx, uint16(addr), values); err != nil {
		return err
	}
	outputSuccess("Wrote %d registers starting at address %d", len(values), addr)
	return nil
}

func (s *InteractiveSession) scanUnits(args []string) error {
	start, end := 1, 247
	if len(args) >= 1 {
		start, _ = strconv.Atoi(args[0])
	}
	if len(args) >= 2 {
		end, _ = strconv.Atoi(args[1])
	}

	fmt.Printf("Scanning unit IDs %d-%d on %s...\n", start, end, s.currentHost)

	found := 0
	for uid := start; uid <= end; uid++ {
		client, err := modbus.NewClient(
			s.currentHost,
			modbus.WithUnitID(modbus.UnitID(uid)),
			modbus.WithTimeout(500*time.Millisecond),
		)
		if err != nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		if err := client.Connect(ctx); err == nil {
			_, err := client.ReadHoldingRegisters(ctx, 0, 1)
			if err == nil || strings.Contains(err.Error(), "exception") {
				fmt.Printf("  Unit %d: %s\n", uid, color(colorGreen, "FOUND"))
				found++
			}
		}
		cancel()
		client.Close()
	}

	fmt.Printf("\nFound %d active unit(s)\n", found)
	return nil
}

func (s *InteractiveSession) getServerID() error {
	if err := s.requireConnection(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	data, err := s.client.ReportServerID(ctx)
	if err != nil {
		return err
	}

	fmt.Println("\nServer Identification:")
	fmt.Println(strings.Repeat("-", 30))
	fmt.Printf("Raw (hex): % X\n", data)
	fmt.Printf("ASCII:     %s\n", strings.TrimRight(string(data), "\x00"))
	fmt.Println()
	return nil
}
