# edgeo-modbus - Modbus TCP Command Line Interface

A comprehensive command-line tool for interacting with Modbus TCP devices.

## Installation

```bash
go build -o edgeo-modbus ./cmd/edgeo-modbus
```

## Commands Overview

| Command | Alias | Description |
|---------|-------|-------------|
| `read` | `r` | Read coils, discrete inputs, or registers |
| `write` | `w` | Write coils or registers |
| `scan` | | Scan for devices, unit IDs, or registers |
| `watch` | | Continuously monitor values |
| `interactive` | `i`, `repl`, `shell` | Interactive REPL shell |
| `diag` | | Diagnostic functions (FC07, FC08, FC11, FC17) |
| `info` | `probe`, `ping` | Get device information |
| `dump` | | Dump register or coil ranges |

## Global Flags

### Connection

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--host` | `-H` | `localhost` | Modbus server host |
| `--port` | `-p` | `502` | Modbus server port |
| `--unit` | `-u` | `1` | Modbus unit ID (1-247) |
| `--timeout` | `-t` | `5s` | Operation timeout |
| `--retries` | `-r` | `3` | Number of retries on failure |

### Output

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `table` | Format: table, json, csv, hex, raw |
| `--verbose` | `-v` | `false` | Verbose output |
| `--no-color` | | `false` | Disable color output |

### Data Format

| Flag | Default | Description |
|------|---------|-------------|
| `--byte-order` | `big` | Byte order: big, little |
| `--word-order` | `big` | Word order for 32-bit values: big, little |

### Configuration

| Flag | Description |
|------|-------------|
| `--config` | Config file path (default: `~/.edgeo-modbus.yaml`) |

## Command: read

Read coils, discrete inputs, holding registers, or input registers from a Modbus device.

### Subcommands

| Subcommand | Aliases | Function Code | Description |
|------------|---------|---------------|-------------|
| `coils` | `c`, `coil` | FC01 | Read coils (discrete outputs) |
| `discrete-inputs` | `di`, `discrete` | FC02 | Read discrete inputs |
| `holding-registers` | `hr`, `holding` | FC03 | Read holding registers |
| `input-registers` | `ir`, `input` | FC04 | Read input registers |

### Usage

```bash
edgeo-modbus read <subcommand> [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--address` | `-a` | `0` | Starting address |
| `--count` | `-c` | `1` | Number of items to read |
| `--format` | `-f` | `uint16` | Data format (registers only) |

### Data Formats (registers only)

| Format | Description |
|--------|-------------|
| `uint16` | Unsigned 16-bit integer (default) |
| `int16` | Signed 16-bit integer |
| `uint32` | Unsigned 32-bit integer (2 registers) |
| `int32` | Signed 32-bit integer (2 registers) |
| `float32` | 32-bit floating point (2 registers) |
| `float64` | 64-bit floating point (4 registers) |
| `string` | ASCII string |

### Examples

```bash
# Read 10 holding registers starting at address 0
edgeo-modbus read hr -a 0 -c 10 -H 192.168.1.100

# Read holding registers as float32 values
edgeo-modbus r hr -a 100 -c 4 -f float32

# Read holding registers as ASCII string
edgeo-modbus r hr -a 0 -c 20 -f string

# Read 10 coils starting at address 0
edgeo-modbus read coils -a 0 -c 10 -H 192.168.1.100

# Read 8 discrete inputs
edgeo-modbus read di -a 100 -c 8

# Read input registers as signed 32-bit integers
edgeo-modbus r ir -a 100 -c 4 -f int32

# JSON output
edgeo-modbus read hr -a 0 -c 5 -o json -H 192.168.1.100
```

## Command: write

Write coils or registers to a Modbus device.

### Subcommands

| Subcommand | Aliases | Function Code | Description |
|------------|---------|---------------|-------------|
| `coil` | `c` | FC05 | Write single coil |
| `coils` | `cs` | FC15 | Write multiple coils |
| `register` | `reg`, `r` | FC06 | Write single register |
| `registers` | `regs`, `rs` | FC16 | Write multiple registers |

### Usage

```bash
edgeo-modbus write <subcommand> [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--address` | `-a` | `0` | Starting address |
| `--values` | `-V` | | Values to write (required) |

### Value Formats

**Coil values** can be: `1`, `0`, `true`, `false`, `on`, `off`, `yes`, `no`

**Register values** can be:
- Decimal: `1234`
- Hexadecimal: `0xFF00`
- Binary: `0b1010101010101010`

Multiple values can be comma-separated or space-separated.

### Examples

```bash
# Write single coil ON
edgeo-modbus write coil -a 0 -V 1 -H 192.168.1.100
edgeo-modbus w c -a 100 -V on

# Write multiple coils
edgeo-modbus write coils -a 0 -V 1,0,1,1,0 -H 192.168.1.100
edgeo-modbus w cs -a 100 -V "1 0 1 1"

# Write single register
edgeo-modbus write register -a 0 -V 1234 -H 192.168.1.100
edgeo-modbus w r -a 100 -V 0xFF00

# Write single register in binary
edgeo-modbus w r -a 50 -V 0b1010101010101010

# Write multiple registers
edgeo-modbus write registers -a 0 -V 100,200,300 -H 192.168.1.100
edgeo-modbus w rs -a 100 -V "0x1234 0x5678"
```

## Command: scan

Scan for Modbus devices on the network or detect active unit IDs and registers.

### Usage

```bash
edgeo-modbus scan [flags]
```

### Scan Types

| Type | Description |
|------|-------------|
| `units` | Scan for active unit IDs on a single host (default) |
| `network` | Scan network range for Modbus devices |
| `registers` | Scan address ranges to find used registers |

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--type` | | `units` | Scan type: units, network, registers |
| `--start-unit` | | `1` | Start unit ID for unit scanning |
| `--end-unit` | | `247` | End unit ID for unit scanning |
| `--start-addr` | `-a` | `0` | Start address for register scanning |
| `--end-addr` | `-e` | `100` | End address for register scanning |
| `--workers` | | `10` | Number of concurrent workers |
| `--scan-timeout` | | `1s` | Timeout for each scan attempt |
| `--network` | | | Network CIDR for network scan (e.g., `192.168.1.0/24`) |
| `--port-start` | | `502` | Start port for network scan |
| `--port-end` | | `502` | End port for network scan |

### Examples

```bash
# Scan for active unit IDs on a host
edgeo-modbus scan -H 192.168.1.100

# Scan specific unit ID range
edgeo-modbus scan --start-unit 1 --end-unit 10 -H 192.168.1.100

# Scan network for Modbus devices
edgeo-modbus scan --type network --network 192.168.1.0/24

# Scan for used registers
edgeo-modbus scan --type registers -a 0 -e 1000 -H 192.168.1.100

# JSON output
edgeo-modbus scan -H 192.168.1.100 -o json
```

**Sample output (unit scan):**

```
Unit Scan Results
----------------------------------------------------------------------
ADDRESS              UNIT ID  LATENCY  SERVER ID   STATUS
-------              -------  -------  ---------   ------
192.168.1.100:502    1        12ms     PLC-01      ONLINE
192.168.1.100:502    2        15ms     PLC-02      ONLINE

Found 2 responsive device(s)
```

## Command: watch

Continuously monitor Modbus registers or coils with configurable interval.

### Subcommands

| Subcommand | Aliases | Description |
|------------|---------|-------------|
| `holding-registers` | `hr`, `holding` | Watch holding registers |
| `input-registers` | `ir`, `input` | Watch input registers |
| `coils` | `c`, `coil` | Watch coils |
| `discrete-inputs` | `di`, `discrete` | Watch discrete inputs |

### Usage

```bash
edgeo-modbus watch <subcommand> [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--address` | `-a` | `0` | Starting address |
| `--count` | `-c` | `1` | Number of items to read |
| `--interval` | `-i` | `1s` | Poll interval |
| `--iterations` | `-n` | `0` | Number of iterations (0 = infinite) |
| `--diff` | | `false` | Highlight changed values |
| `--clear` | | `true` | Clear terminal between updates |
| `--timestamp` | | `true` | Show timestamps |
| `--log` | | | Log values to file (CSV format) |

**Register-specific flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `uint16` | Data format |
| `--alert-high` | `0` | Alert when value exceeds this threshold |
| `--alert-low` | `0` | Alert when value falls below this threshold |
| `--alert` | `false` | Enable threshold alerts |

### Examples

```bash
# Watch 5 holding registers every second
edgeo-modbus watch hr -a 0 -c 5 -i 1s -H 192.168.1.100

# Watch with alerts when value exceeds threshold
edgeo-modbus watch hr -a 100 -c 1 -i 500ms --alert-high 1000 --alert

# Watch and log to file
edgeo-modbus watch hr -a 0 -c 10 -i 2s --log data.csv

# Watch coils with change highlighting
edgeo-modbus watch c -a 0 -c 8 -i 1s --diff

# Watch for 10 iterations only
edgeo-modbus watch hr -a 0 -c 5 -n 10 -H 192.168.1.100

# Watch in JSON output mode (one JSON object per iteration)
edgeo-modbus watch hr -a 0 -c 5 -i 1s -o json
```

**Sample output:**

```
MODBUS WATCH - Watching Holding Registers (Address 0-4)
Host: 192.168.1.100:502 | Unit: 1 | Interval: 1s
Time: 14:32:15.123 | Iteration: 5
------------------------------------------------------------
ADDR  VALUE  HEX     CHANGE
----  -----  ---     ------
0     1234   0x04D2
1     5678   0x162E  +12
2     0      0x0000
3     255    0x00FF  -3
4     9999   0x270F
```

## Command: interactive

Start an interactive REPL session for Modbus communication.

### Usage

```bash
edgeo-modbus interactive [flags]
```

### Shell Commands

| Command | Description |
|---------|-------------|
| **Connection** | |
| `connect [host:port]` | Connect to Modbus server |
| `disconnect` | Disconnect from server |
| `unit <id>` | Set/show unit ID (1-247) |
| `status` | Show connection status |
| **Read Operations** | |
| `rc <addr> [count]` | Read coils (FC01) |
| `rdi <addr> [count]` | Read discrete inputs (FC02) |
| `rhr <addr> [count] [format]` | Read holding registers (FC03) |
| `rir <addr> [count] [format]` | Read input registers (FC04) |
| **Write Operations** | |
| `wc <addr> <0\|1>` | Write single coil (FC05) |
| `wr <addr> <value>` | Write single register (FC06) |
| `wcs <addr> <v1,v2,...>` | Write multiple coils (FC15) |
| `wrs <addr> <v1,v2,...>` | Write multiple registers (FC16) |
| **Tools** | |
| `scan [start] [end]` | Scan for active unit IDs |
| `id` | Get server identification |
| **Settings** | |
| `output <format>` | Set output format (table/json/csv/hex/raw) |
| `format <type>` | Set register format (uint16/int16/uint32/int32/float32) |
| **General** | |
| `help` | Show help |
| `quit` / `exit` | Exit interactive mode |

### Example Session

```
$ edgeo-modbus interactive -H 192.168.1.100
Modbus Interactive Shell
Type 'help' for available commands, 'quit' to exit

modbus[192.168.1.100:502]@1> status

Connection Status
------------------------------
Status:        Connected
Host:          192.168.1.100:502
Unit ID:       1
Output:        table
Reg Format:    uint16
Timeout:       5s

modbus[192.168.1.100:502]@1> rhr 0 5
ADDR  VALUE  HEX
0     1234   0x04D2
1     5678   0x162E
2     0      0x0000
3     255    0x00FF
4     9999   0x270F

modbus[192.168.1.100:502]@1> wr 0 5000
Wrote register 0 = 5000 (0x1388)

modbus[192.168.1.100:502]@1> wc 0 1
Wrote coil 0 = true

modbus[192.168.1.100:502]@1> unit 2
Unit ID set to 2

modbus[192.168.1.100:502]@2> scan 1 10
Scanning unit IDs 1-10 on 192.168.1.100:502...
  Unit 1: FOUND
  Unit 2: FOUND

Found 2 active unit(s)

modbus[192.168.1.100:502]@2> quit
Goodbye!
```

## Command: diag

Execute Modbus diagnostic functions.

### Subcommands

| Subcommand | Aliases | Function Code | Description |
|------------|---------|---------------|-------------|
| `exception-status` | `es`, `exception` | FC07 | Read exception status |
| `diagnostics` | `d` | FC08 | Execute diagnostics |
| `comm-event-counter` | `cec`, `events` | FC11 | Get communication event counter |
| `server-id` | `id` | FC17 | Report server ID |

### Usage

```bash
edgeo-modbus diag <subcommand> [flags]
```

### Diagnostics Sub-functions (FC08)

| Code | Name |
|------|------|
| 0 | Return Query Data (echo test) |
| 1 | Restart Communications |
| 10 | Clear Counters |
| 11 | Return Bus Message Count |
| 12 | Return Bus Communication Error Count |
| 13 | Return Bus Exception Error Count |
| 14 | Return Server Message Count |
| 15 | Return Server No Response Count |
| 16 | Return Server NAK Count |
| 17 | Return Server Busy Count |
| 18 | Return Bus Character Overrun Count |

### Diagnostics Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--subfunc` | `-s` | `0` | Sub-function code |
| `--data` | `-d` | | Data to send (for echo test) |

### Examples

```bash
# Read exception status
edgeo-modbus diag exception-status -H 192.168.1.100

# Run echo test (sub-function 0)
edgeo-modbus diag diagnostics -s 0 -d "Hello" -H 192.168.1.100

# Get bus message count (sub-function 11)
edgeo-modbus diag diagnostics -s 11 -H 192.168.1.100

# Get communication event counter
edgeo-modbus diag comm-event-counter -H 192.168.1.100

# Report server ID
edgeo-modbus diag server-id -H 192.168.1.100

# JSON output
edgeo-modbus diag server-id -H 192.168.1.100 -o json
```

**Sample output (server-id):**

```
Server Identification (FC17)
----------------------------------------
Length:  12 bytes
Raw:     01 FF 50 4C 43 2D 30 31 00 00 00 00
ASCII:   PLC-01

Decoded fields:
  Server ID:   1
  Run Status:  ON
  Additional:  50 4C 43 2D 30 31 00 00 00 00
  (as text):   PLC-01
```

## Command: info

Probe a Modbus device and retrieve all available information.

### Usage

```bash
edgeo-modbus info [flags]
```

### Information Retrieved

The `info` command performs the following checks:

1. Tests TCP connectivity
2. Attempts Modbus connection
3. Measures response latency
4. Reads server identification (FC17)
5. Reads exception status (FC07)
6. Tests supported function codes

### Examples

```bash
# Get device info
edgeo-modbus info -H 192.168.1.100

# Get info for specific unit ID
edgeo-modbus info -H 10.0.0.50 -u 2

# JSON output
edgeo-modbus info -H 192.168.1.100 -o json
```

**Sample output:**

```
Device Information
==================================================
Address:      192.168.1.100:502
Unit ID:      1
TCP:          Reachable
Modbus:       Connected
Latency:      12ms
Server ID:    PLC-01
Run Status:   Running
Exception:    0x00 (00000000)

Supported Functions:
  OK Read Coils (FC01)
  OK Read Discrete Inputs (FC02)
  OK Read Holding Registers (FC03)
  OK Read Input Registers (FC04)
```

## Command: dump

Dump a range of registers or coils from the Modbus device with export support.

### Subcommands

| Subcommand | Aliases | Description |
|------------|---------|-------------|
| `holding-registers` | `hr`, `holding` | Dump holding registers |
| `input-registers` | `ir`, `input` | Dump input registers |
| `coils` | `c`, `coil` | Dump coils |
| `discrete-inputs` | `di`, `discrete` | Dump discrete inputs |

### Usage

```bash
edgeo-modbus dump <subcommand> [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--start` | `-a` | `0` | Start address |
| `--end` | `-e` | `100` | End address |
| `--batch` | `-b` | `125` (regs) / `2000` (coils) | Batch size for reading |
| `--file` | `-f` | | Output file (default: stdout) |
| `--show-empty` | | `false` | Show addresses that return errors |

### Examples

```bash
# Dump holding registers to console
edgeo-modbus dump hr -a 0 -e 999 -H 192.168.1.100

# Dump input registers to CSV file
edgeo-modbus dump ir -a 0 -e 100 -f registers.csv -o csv

# Dump to JSON file
edgeo-modbus dump hr -a 0 -e 100 -f dump.json -o json

# Dump coils
edgeo-modbus dump coils -a 0 -e 100 -H 192.168.1.100

# Dump with hex output
edgeo-modbus dump hr -a 0 -e 255 -o hex

# Include error addresses in output
edgeo-modbus dump hr -a 0 -e 500 --show-empty
```

**Sample output (table):**

```
Holding Registers Dump
============================================================
    0:  04D2  162E  0000  00FF  270F  0000  0000  0000
    8:  FFFF  0001  0002  0003  0004  0005  0006  0007
```

**Sample output (hex):**

```
00000000  04 d2 16 2e 00 00 00 ff 27 0f 00 00 00 00 00 00  |...........'....|
00000008  ff ff 00 01 00 02 00 03 00 04 00 05 00 06 00 07  |................|
```

## Configuration File

Create `~/.edgeo-modbus.yaml` for default settings:

```yaml
# Connection
host: localhost
port: 502
unit: 1
timeout: 5s
retries: 3

# Data Format
byte-order: big
word-order: big

# Output
output: table
verbose: false
```

## Environment Variables

Environment variables use the `MODBUS_` prefix:

```bash
export MODBUS_HOST=192.168.1.100
export MODBUS_PORT=502
export MODBUS_TIMEOUT=5s
```

## Output Formats

### Table (default)

```
ADDR  VALUE  HEX
0     1234   0x04D2
1     5678   0x162E
2     0      0x0000
```

### JSON

```json
[
  {
    "address": 0,
    "value": 1234,
    "hex": "0x04D2"
  },
  {
    "address": 1,
    "value": 5678,
    "hex": "0x162E"
  }
]
```

### CSV

```csv
address,value,hex,error
0,1234,0x04D2,
1,5678,0x162E,
```

### Hex

```
00000000  04 d2 16 2e 00 00 00 ff  |........|
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Error (connection failed, read/write failed, etc.) |

## See Also

- [Client Library Documentation](client.md)
- [Server Documentation](server.md)
- [Configuration Options](options.md)
