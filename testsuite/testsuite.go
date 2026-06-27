// Package testsuite esegue i vettori per-istruzione SingleStepTests (TomHarte)
// per l'8088 contro il core di retronet-8086. Ogni vettore descrive lo stato
// completo di CPU e RAM prima e dopo una singola istruzione: il loader imposta lo
// stato iniziale, esegue uno Step e confronta lo stato finale.
//
// I dataset NON sono inclusi nel repo: vanno scaricati a parte e passati come
// directory. I file possono essere .json o .json.gz (un file per opcode).
package testsuite

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/retronet-labs/retronet-8086/cpu"
)

// vector e' un singolo caso di test del formato SingleStepTests.
type vector struct {
	Name    string `json:"name"`
	Initial state  `json:"initial"`
	Final   state  `json:"final"`
}

type state struct {
	Regs map[string]int `json:"regs"`
	RAM  [][]int        `json:"ram"` // coppie [indirizzo, valore]
}

// Options regola l'esecuzione della suite.
type Options struct {
	Backend cpu.ALUBackend // Gate (default) o Native
	// FlagMask azzera (ignora) i bit di flag indicati nel confronto: serve per i
	// flag lasciati indefiniti dall'8086 in alcune istruzioni. 0 = confronto stretto.
	FlagMask uint16
	// MaxPerFile limita i vettori per file (0 = tutti). Utile per smoke test rapidi.
	MaxPerFile int
	// StopOnFirstFailure interrompe un file al primo fallimento.
	StopOnFirstFailure bool
}

// Result riassume l'esito di una o piu' esecuzioni.
type Result struct {
	Total    int
	Passed   int
	Failures []string
}

// Failed restituisce i vettori non passati.
func (r Result) Failed() int { return r.Total - r.Passed }

// RunDir esegue tutti i file .json/.json.gz nella directory (ordinati per nome).
func RunDir(dir string, opt Options) (Result, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Result{}, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".json") || strings.HasSuffix(n, ".json.gz") {
			names = append(names, n)
		}
	}
	sort.Strings(names)

	var total Result
	for _, n := range names {
		r, err := RunFile(filepath.Join(dir, n), opt)
		if err != nil {
			return total, fmt.Errorf("%s: %w", n, err)
		}
		total.Total += r.Total
		total.Passed += r.Passed
		total.Failures = append(total.Failures, r.Failures...)
	}
	return total, nil
}

// RunFile esegue un singolo file di vettori.
func RunFile(path string, opt Options) (Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()

	var reader io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return Result{}, err
		}
		defer gz.Close()
		reader = gz
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return Result{}, err
	}
	return RunBytes(data, opt)
}

// RunBytes esegue i vettori contenuti in un documento JSON (array di casi).
func RunBytes(data []byte, opt Options) (Result, error) {
	var vectors []vector
	if err := json.Unmarshal(data, &vectors); err != nil {
		return Result{}, err
	}
	backend := opt.Backend
	if backend == nil {
		backend = cpu.Gate
	}

	var r Result
	for i, v := range vectors {
		if opt.MaxPerFile > 0 && i >= opt.MaxPerFile {
			break
		}
		r.Total++
		if msg, ok := runVector(v, backend, opt.FlagMask); ok {
			r.Passed++
		} else {
			r.Failures = append(r.Failures, fmt.Sprintf("%s: %s", v.Name, msg))
			if opt.StopOnFirstFailure {
				break
			}
		}
	}
	return r, nil
}

// runVector imposta lo stato iniziale, esegue uno Step e confronta col finale.
func runVector(v vector, backend cpu.ALUBackend, flagMask uint16) (string, bool) {
	c := cpu.NewCPU8086WithALU(backend)
	applyRegs(c, v.Initial.Regs)
	applyRAM(c, v.Initial.RAM)

	if err := c.Step(); err != nil {
		return "Step: " + err.Error(), false
	}

	if msg, ok := checkRegs(c, v.Final.Regs, flagMask); !ok {
		return msg, false
	}
	return checkRAM(c, v.Final.RAM)
}

func applyRegs(c *cpu.CPU8086, regs map[string]int) {
	for k, val := range regs {
		v := uint16(val)
		switch k {
		case "ax":
			c.Regs[cpu.AX] = v
		case "bx":
			c.Regs[cpu.BX] = v
		case "cx":
			c.Regs[cpu.CX] = v
		case "dx":
			c.Regs[cpu.DX] = v
		case "sp":
			c.Regs[cpu.SP] = v
		case "bp":
			c.Regs[cpu.BP] = v
		case "si":
			c.Regs[cpu.SI] = v
		case "di":
			c.Regs[cpu.DI] = v
		case "cs":
			c.Seg[cpu.CS] = v
		case "ss":
			c.Seg[cpu.SS] = v
		case "ds":
			c.Seg[cpu.DS] = v
		case "es":
			c.Seg[cpu.ES] = v
		case "ip":
			c.IP = v
		case "flags":
			c.SetFlags(v)
		}
	}
}

func applyRAM(c *cpu.CPU8086, ram [][]int) {
	for _, cell := range ram {
		c.Mem.Write8(uint32(cell[0])&cpu.AddressMask, byte(cell[1]))
	}
}

func checkRegs(c *cpu.CPU8086, regs map[string]int, flagMask uint16) (string, bool) {
	actual := map[string]uint16{
		"ax": c.Regs[cpu.AX], "bx": c.Regs[cpu.BX], "cx": c.Regs[cpu.CX], "dx": c.Regs[cpu.DX],
		"sp": c.Regs[cpu.SP], "bp": c.Regs[cpu.BP], "si": c.Regs[cpu.SI], "di": c.Regs[cpu.DI],
		"cs": c.Seg[cpu.CS], "ss": c.Seg[cpu.SS], "ds": c.Seg[cpu.DS], "es": c.Seg[cpu.ES],
		"ip": c.IP, "flags": c.PackFlags(),
	}
	for k, want := range regs {
		got, ok := actual[k]
		if !ok {
			continue
		}
		w := uint16(want)
		if k == "flags" {
			if got&^flagMask != w&^flagMask {
				return fmt.Sprintf("flags: got %#04x want %#04x (mask %#04x)", got, w, flagMask), false
			}
			continue
		}
		if got != w {
			return fmt.Sprintf("%s: got %#04x want %#04x", k, got, w), false
		}
	}
	return "", true
}

func checkRAM(c *cpu.CPU8086, ram [][]int) (string, bool) {
	for _, cell := range ram {
		addr := uint32(cell[0]) & cpu.AddressMask
		want := byte(cell[1])
		if got := c.Mem.Read8(addr); got != want {
			return fmt.Sprintf("ram[%#05x]: got %#02x want %#02x", addr, got, want), false
		}
	}
	return "", true
}
