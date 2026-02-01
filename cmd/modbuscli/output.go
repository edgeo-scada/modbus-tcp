package main

import (
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
)

// Color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

func color(c, s string) string {
	if noColor {
		return s
	}
	return c + s + colorReset
}

func outputSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(color(colorGreen, "OK") + " " + msg)
}

func outputError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, color(colorRed, "ERROR")+" "+msg)
}

func outputWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, color(colorYellow, "WARN")+" "+msg)
}

func outputInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(color(colorCyan, "INFO")+" "+msg)
}

type BoolResult struct {
	Address uint16 `json:"address"`
	Value   bool   `json:"value"`
}

type RegisterResult struct {
	Address uint16      `json:"address"`
	Raw     uint16      `json:"raw"`
	Hex     string      `json:"hex"`
	Value   interface{} `json:"value,omitempty"`
	Format  string      `json:"format,omitempty"`
}

func outputBoolValues(title string, startAddr uint16, values []bool) error {
	switch outputFmt {
	case "json":
		return outputBoolJSON(startAddr, values)
	case "csv":
		return outputBoolCSV(startAddr, values)
	case "raw":
		return outputBoolRaw(values)
	case "hex":
		return outputBoolHex(startAddr, values)
	default:
		return outputBoolTable(title, startAddr, values)
	}
}

func outputBoolTable(title string, startAddr uint16, values []bool) error {
	fmt.Printf("\n%s (Address %d-%d, Count: %d)\n",
		color(colorBold, title),
		startAddr,
		startAddr+uint16(len(values))-1,
		len(values))
	fmt.Println(strings.Repeat("-", 40))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ADDRESS\tVALUE\tSTATUS")
	fmt.Fprintln(w, "-------\t-----\t------")

	for i, v := range values {
		addr := startAddr + uint16(i)
		var valStr, statusStr string
		if v {
			valStr = "1"
			statusStr = color(colorGreen, "ON")
		} else {
			valStr = "0"
			statusStr = color(colorRed, "OFF")
		}
		fmt.Fprintf(w, "%d\t%s\t%s\n", addr, valStr, statusStr)
	}
	w.Flush()
	fmt.Println()
	return nil
}

func outputBoolJSON(startAddr uint16, values []bool) error {
	results := make([]BoolResult, len(values))
	for i, v := range values {
		results[i] = BoolResult{
			Address: startAddr + uint16(i),
			Value:   v,
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func outputBoolCSV(startAddr uint16, values []bool) error {
	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"address", "value"})
	for i, v := range values {
		addr := strconv.Itoa(int(startAddr) + i)
		val := "0"
		if v {
			val = "1"
		}
		w.Write([]string{addr, val})
	}
	w.Flush()
	return w.Error()
}

func outputBoolRaw(values []bool) error {
	for _, v := range values {
		if v {
			fmt.Print("1")
		} else {
			fmt.Print("0")
		}
	}
	fmt.Println()
	return nil
}

func outputBoolHex(startAddr uint16, values []bool) error {
	// Pack bools into bytes
	byteCount := (len(values) + 7) / 8
	bytes := make([]byte, byteCount)
	for i, v := range values {
		if v {
			bytes[i/8] |= 1 << (i % 8)
		}
	}
	for i, b := range bytes {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%02X", b)
	}
	fmt.Println()
	return nil
}

func outputRegisterValues(title string, startAddr uint16, values []uint16, format string) error {
	switch outputFmt {
	case "json":
		return outputRegisterJSON(startAddr, values, format)
	case "csv":
		return outputRegisterCSV(startAddr, values, format)
	case "raw":
		return outputRegisterRaw(values)
	case "hex":
		return outputRegisterHex(values)
	default:
		return outputRegisterTable(title, startAddr, values, format)
	}
}

func outputRegisterTable(title string, startAddr uint16, values []uint16, format string) error {
	fmt.Printf("\n%s (Address %d-%d, Count: %d)\n",
		color(colorBold, title),
		startAddr,
		startAddr+uint16(len(values))-1,
		len(values))
	fmt.Println(strings.Repeat("-", 60))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	switch format {
	case "uint16", "":
		fmt.Fprintln(w, "ADDRESS\tDECIMAL\tHEX\tBINARY")
		fmt.Fprintln(w, "-------\t-------\t---\t------")
		for i, v := range values {
			addr := startAddr + uint16(i)
			fmt.Fprintf(w, "%d\t%d\t0x%04X\t%016b\n", addr, v, v, v)
		}

	case "int16":
		fmt.Fprintln(w, "ADDRESS\tDECIMAL\tHEX")
		fmt.Fprintln(w, "-------\t-------\t---")
		for i, v := range values {
			addr := startAddr + uint16(i)
			signed := int16(v)
			fmt.Fprintf(w, "%d\t%d\t0x%04X\n", addr, signed, v)
		}

	case "uint32":
		fmt.Fprintln(w, "ADDRESS\tVALUE\tHEX")
		fmt.Fprintln(w, "-------\t-----\t---")
		for i := 0; i < len(values)-1; i += 2 {
			addr := startAddr + uint16(i)
			val := combineRegisters(values[i], values[i+1])
			fmt.Fprintf(w, "%d-%d\t%d\t0x%08X\n", addr, addr+1, val, val)
		}

	case "int32":
		fmt.Fprintln(w, "ADDRESS\tVALUE\tHEX")
		fmt.Fprintln(w, "-------\t-----\t---")
		for i := 0; i < len(values)-1; i += 2 {
			addr := startAddr + uint16(i)
			val := int32(combineRegisters(values[i], values[i+1]))
			fmt.Fprintf(w, "%d-%d\t%d\t0x%08X\n", addr, addr+1, val, uint32(val))
		}

	case "float32":
		fmt.Fprintln(w, "ADDRESS\tVALUE\tHEX")
		fmt.Fprintln(w, "-------\t-----\t---")
		for i := 0; i < len(values)-1; i += 2 {
			addr := startAddr + uint16(i)
			bits := combineRegisters(values[i], values[i+1])
			val := math.Float32frombits(bits)
			fmt.Fprintf(w, "%d-%d\t%g\t0x%08X\n", addr, addr+1, val, bits)
		}

	case "float64":
		fmt.Fprintln(w, "ADDRESS\tVALUE\tHEX")
		fmt.Fprintln(w, "-------\t-----\t---")
		for i := 0; i < len(values)-3; i += 4 {
			addr := startAddr + uint16(i)
			bits := combineRegisters64(values[i], values[i+1], values[i+2], values[i+3])
			val := math.Float64frombits(bits)
			fmt.Fprintf(w, "%d-%d\t%g\t0x%016X\n", addr, addr+3, val, bits)
		}

	case "string":
		fmt.Fprintln(w, "STRING VALUE:")
		var sb strings.Builder
		for _, v := range values {
			// Big-endian byte order for ASCII
			sb.WriteByte(byte(v >> 8))
			sb.WriteByte(byte(v & 0xFF))
		}
		fmt.Fprintln(w, strings.TrimRight(sb.String(), "\x00"))
	}

	w.Flush()
	fmt.Println()
	return nil
}

func outputRegisterJSON(startAddr uint16, values []uint16, format string) error {
	results := make([]RegisterResult, 0)

	switch format {
	case "uint32":
		for i := 0; i < len(values)-1; i += 2 {
			val := combineRegisters(values[i], values[i+1])
			results = append(results, RegisterResult{
				Address: startAddr + uint16(i),
				Raw:     values[i],
				Hex:     fmt.Sprintf("0x%08X", val),
				Value:   val,
				Format:  format,
			})
		}
	case "int32":
		for i := 0; i < len(values)-1; i += 2 {
			val := int32(combineRegisters(values[i], values[i+1]))
			results = append(results, RegisterResult{
				Address: startAddr + uint16(i),
				Raw:     values[i],
				Hex:     fmt.Sprintf("0x%08X", uint32(val)),
				Value:   val,
				Format:  format,
			})
		}
	case "float32":
		for i := 0; i < len(values)-1; i += 2 {
			bits := combineRegisters(values[i], values[i+1])
			val := math.Float32frombits(bits)
			results = append(results, RegisterResult{
				Address: startAddr + uint16(i),
				Raw:     values[i],
				Hex:     fmt.Sprintf("0x%08X", bits),
				Value:   val,
				Format:  format,
			})
		}
	case "int16":
		for i, v := range values {
			results = append(results, RegisterResult{
				Address: startAddr + uint16(i),
				Raw:     v,
				Hex:     fmt.Sprintf("0x%04X", v),
				Value:   int16(v),
				Format:  format,
			})
		}
	default: // uint16
		for i, v := range values {
			results = append(results, RegisterResult{
				Address: startAddr + uint16(i),
				Raw:     v,
				Hex:     fmt.Sprintf("0x%04X", v),
				Value:   v,
				Format:  "uint16",
			})
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func outputRegisterCSV(startAddr uint16, values []uint16, format string) error {
	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"address", "raw", "hex", "value"})

	switch format {
	case "uint32":
		for i := 0; i < len(values)-1; i += 2 {
			addr := strconv.Itoa(int(startAddr) + i)
			val := combineRegisters(values[i], values[i+1])
			w.Write([]string{addr, strconv.Itoa(int(values[i])), fmt.Sprintf("0x%08X", val), strconv.FormatUint(uint64(val), 10)})
		}
	case "int32":
		for i := 0; i < len(values)-1; i += 2 {
			addr := strconv.Itoa(int(startAddr) + i)
			val := int32(combineRegisters(values[i], values[i+1]))
			w.Write([]string{addr, strconv.Itoa(int(values[i])), fmt.Sprintf("0x%08X", uint32(val)), strconv.Itoa(int(val))})
		}
	case "float32":
		for i := 0; i < len(values)-1; i += 2 {
			addr := strconv.Itoa(int(startAddr) + i)
			bits := combineRegisters(values[i], values[i+1])
			val := math.Float32frombits(bits)
			w.Write([]string{addr, strconv.Itoa(int(values[i])), fmt.Sprintf("0x%08X", bits), fmt.Sprintf("%g", val)})
		}
	default:
		for i, v := range values {
			addr := strconv.Itoa(int(startAddr) + i)
			w.Write([]string{addr, strconv.Itoa(int(v)), fmt.Sprintf("0x%04X", v), strconv.Itoa(int(v))})
		}
	}

	w.Flush()
	return w.Error()
}

func outputRegisterRaw(values []uint16) error {
	for _, v := range values {
		fmt.Printf("%d\n", v)
	}
	return nil
}

func outputRegisterHex(values []uint16) error {
	for i, v := range values {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%04X", v)
	}
	fmt.Println()
	return nil
}

func combineRegisters(high, low uint16) uint32 {
	if wordOrder == "little" {
		return uint32(low)<<16 | uint32(high)
	}
	return uint32(high)<<16 | uint32(low)
}

func combineRegisters64(r0, r1, r2, r3 uint16) uint64 {
	if wordOrder == "little" {
		return uint64(r3)<<48 | uint64(r2)<<32 | uint64(r1)<<16 | uint64(r0)
	}
	return uint64(r0)<<48 | uint64(r1)<<32 | uint64(r2)<<16 | uint64(r3)
}

func swapBytes(v uint16) uint16 {
	return (v >> 8) | (v << 8)
}

func uint16ToBytes(values []uint16) []byte {
	result := make([]byte, len(values)*2)
	for i, v := range values {
		if byteOrder == "little" {
			binary.LittleEndian.PutUint16(result[i*2:], v)
		} else {
			binary.BigEndian.PutUint16(result[i*2:], v)
		}
	}
	return result
}
