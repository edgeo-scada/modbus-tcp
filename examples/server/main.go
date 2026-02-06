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
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/edgeo-scada/modbus"
)

func main() {
	addr := flag.String("addr", ":502", "Server address")
	flag.Parse()

	// Setup logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create memory handler with some initial data
	handler := modbus.NewMemoryHandler(65536, 65536)

	// Set some initial values for testing
	unitID := modbus.UnitID(1)

	// Set coils
	handler.SetCoil(unitID, 0, true)
	handler.SetCoil(unitID, 1, false)
	handler.SetCoil(unitID, 2, true)

	// Set discrete inputs
	handler.SetDiscreteInput(unitID, 0, true)
	handler.SetDiscreteInput(unitID, 1, true)

	// Set holding registers
	handler.SetHoldingRegister(unitID, 0, 1234)
	handler.SetHoldingRegister(unitID, 1, 5678)
	handler.SetHoldingRegister(unitID, 2, 9012)

	// Set input registers
	handler.SetInputRegister(unitID, 0, 100)
	handler.SetInputRegister(unitID, 1, 200)

	// Set server ID
	handler.SetServerID([]byte("Edgeo Modbus Server v1.0"))

	// Create server
	server := modbus.NewServer(handler,
		modbus.WithServerLogger(logger),
		modbus.WithMaxConnections(10),
	)

	// Handle shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
		server.Close()
	}()

	// Start server
	fmt.Printf("Starting Modbus TCP server on %s\n", *addr)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("\nInitial data:")
	fmt.Printf("  Coils[0:3]: true, false, true\n")
	fmt.Printf("  Discrete Inputs[0:2]: true, true\n")
	fmt.Printf("  Holding Registers[0:3]: 1234, 5678, 9012\n")
	fmt.Printf("  Input Registers[0:2]: 100, 200\n")
	fmt.Println()

	if err := server.ListenAndServeContext(ctx, *addr); err != nil {
		logger.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
