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
	Bytes   []int  `json:"bytes"`
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

	// Una sola CPU riusata per tutti i vettori: la RAM da 1 MB si alloca una volta
	// e tra un vettore e l'altro si azzerano solo le celle toccate (i vettori
	// assumono memoria a zero salvo le celle elencate).
	c := cpu.NewCPU8086WithALU(backend)
	var touched []uint32

	var r Result
	for i, v := range vectors {
		if opt.MaxPerFile > 0 && i >= opt.MaxPerFile {
			break
		}
		r.Total++
		if msg, ok := runVector(c, &touched, v, opt.FlagMask); ok {
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

// runVector ripristina la CPU riusata, imposta lo stato iniziale, esegue uno Step
// e confronta col finale. touched accumula le celle scritte da azzerare prima del
// vettore successivo.
func runVector(c *cpu.CPU8086, touched *[]uint32, v vector, flagMask uint16) (string, bool) {
	for _, a := range *touched {
		c.Mem.Write8(a, 0)
	}
	*touched = (*touched)[:0]

	c.Halted = false
	applyRegs(c, v.Initial.Regs) // i vettori v2 specificano tutti i registri
	for _, cell := range v.Initial.RAM {
		a := uint32(cell[0]) & cpu.AddressMask
		c.Mem.Write8(a, byte(cell[1]))
		*touched = append(*touched, a)
	}
	for _, cell := range v.Final.RAM {
		*touched = append(*touched, uint32(cell[0])&cpu.AddressMask)
	}

	if err := c.Step(); err != nil {
		return "Step: " + err.Error(), false
	}
	// Ai flag ignorati richiesti si aggiungono quelli che l'8086 lascia
	// *indefiniti* per questa istruzione (e che il silicio riempie in modo non
	// documentato): vanno esclusi dal confronto.
	mask := flagMask | undefinedFlagMask(v.Bytes)
	if msg, ok := checkRegs(c, v.Final.Regs, mask); !ok {
		return msg, false
	}
	return checkRAM(c, v.Final.RAM)
}

// undefinedFlagMask restituisce i bit di flag lasciati indefiniti dall'8086 per
// l'istruzione codificata in bytes (prefissi inclusi). Sono i bit da NON
// confrontare coi vettori, che li riportano coi valori reali del silicio.
func undefinedFlagMask(bytes []int) uint16 {
	const (
		CF = 1 << 0
		PF = 1 << 2
		AF = 1 << 4
		ZF = 1 << 6
		SF = 1 << 7
		OF = 1 << 11
	)
	i := 0
	for i < len(bytes) && isPrefixByte(byte(bytes[i])) {
		i++
	}
	if i >= len(bytes) {
		return 0
	}
	op := byte(bytes[i])
	var reg byte
	if i+1 < len(bytes) {
		reg = byte(bytes[i+1]) >> 3 & 0x07
	}
	switch op {
	case 0x27, 0x2F: // DAA/DAS: OF indefinito
		return OF
	case 0x37, 0x3F: // AAA/AAS: OF/SF/ZF/PF indefiniti
		return OF | SF | ZF | PF
	case 0xD4, 0xD5: // AAM/AAD: OF/CF/AF indefiniti
		return OF | CF | AF
	case 0xD0, 0xD1: // shift/rotate per 1: AF indefinito (OF definito)
		return AF
	case 0xD2, 0xD3: // shift/rotate per CL: AF e OF indefiniti (OF solo se CL==1)
		return AF | OF
	case 0xF6, 0xF7:
		switch reg {
		case 4, 5: // MUL/IMUL: SF/ZF/AF/PF indefiniti (CF/OF definiti)
			return SF | ZF | AF | PF
		case 6, 7: // DIV/IDIV: tutti i flag aritmetici indefiniti
			return CF | OF | SF | ZF | AF | PF
		}
	}
	return 0
}

func isPrefixByte(b byte) bool {
	switch b {
	case 0x26, 0x2E, 0x36, 0x3E, 0xF0, 0xF2, 0xF3:
		return true
	}
	return false
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
