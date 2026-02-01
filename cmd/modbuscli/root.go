package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	// Global flags
	host       string
	port       int
	unitID     uint8
	timeout    time.Duration
	retries    int
	outputFmt  string
	verbose    bool
	noColor    bool
	byteOrder  string
	wordOrder  string

	logger *slog.Logger
)

var rootCmd = &cobra.Command{
	Use:   "modbuscli",
	Short: "A comprehensive Modbus TCP client CLI",
	Long: `modbuscli is a powerful command-line interface for interacting with Modbus TCP devices.

Features:
  - Read/write coils and registers
  - Multiple output formats (table, json, csv, hex, raw)
  - Device scanning and discovery
  - Continuous monitoring (watch mode)
  - Interactive REPL mode
  - Configuration file support

Examples:
  # Read 10 holding registers from address 0
  modbuscli read hr -a 0 -c 10 -H 192.168.1.100

  # Write value 1234 to register 100
  modbuscli write register -a 100 -v 1234 -H 192.168.1.100

  # Scan for Modbus devices
  modbuscli scan -H 192.168.1.100

  # Interactive mode
  modbuscli interactive -H 192.168.1.100

  # Watch registers continuously
  modbuscli watch hr -a 0 -c 5 -i 1s -H 192.168.1.100`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Setup logger
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		}
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		}))
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	// Configuration file
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.modbuscli.yaml)")

	// Connection flags
	rootCmd.PersistentFlags().StringVarP(&host, "host", "H", "localhost", "Modbus server host")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 502, "Modbus server port")
	rootCmd.PersistentFlags().Uint8VarP(&unitID, "unit", "u", 1, "Modbus unit ID (1-247)")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", 5*time.Second, "Operation timeout")
	rootCmd.PersistentFlags().IntVarP(&retries, "retries", "r", 3, "Number of retries on failure")

	// Output flags
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "Output format: table, json, csv, hex, raw")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")

	// Data format flags
	rootCmd.PersistentFlags().StringVar(&byteOrder, "byte-order", "big", "Byte order: big, little")
	rootCmd.PersistentFlags().StringVar(&wordOrder, "word-order", "big", "Word order for 32-bit values: big, little")

	// Bind to viper
	viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("unit", rootCmd.PersistentFlags().Lookup("unit"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))

	// Add commands
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(writeCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(interactiveCmd)
	rootCmd.AddCommand(diagCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(dumpCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".modbuscli")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("MODBUS")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

func getAddress() string {
	return fmt.Sprintf("%s:%d", viper.GetString("host"), viper.GetInt("port"))
}
