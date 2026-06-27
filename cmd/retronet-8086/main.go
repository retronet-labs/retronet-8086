// Comando retronet-8086: esegue programmi raw sull'emulatore Intel 8086/8088 e
// lancia la suite di conformita' SingleStepTests.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/retronet-labs/retronet-8086/cpu"
	"github.com/retronet-labs/retronet-8086/testsuite"
)

func main() {
	bin := flag.String("bin", "", "file binario raw da caricare ed eseguire")
	steps := flag.Int("steps", 100000, "numero massimo di istruzioni")
	trace := flag.Bool("trace", false, "stampa i registri a ogni istruzione")
	profiles := flag.Bool("profiles", false, "elenca i profili di chip disponibili")
	aluName := flag.String("alu", "gate", "backend ALU: gate oppure native")
	loadSeg := flag.Int("seg", 0x0000, "segmento di caricamento (CS)")
	loadOff := flag.Int("off", 0x0100, "offset di caricamento (IP)")
	suite := flag.String("testsuite", "", "directory dei vettori SingleStepTests/8088 da eseguire")
	flag.Parse()

	switch {
	case *profiles:
		for _, p := range cpu.Profiles() {
			fmt.Printf("%-6s bus dati %d bit, coda prefetch %d byte\n", p.Name, p.DataBusBits, p.PrefetchBytes)
		}
		return
	case *suite != "":
		runSuite(*suite, backendFor(*aluName))
		return
	case *bin != "":
		runBinary(*bin, backendFor(*aluName), uint16(*loadSeg), uint16(*loadOff), *steps, *trace)
		return
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func backendFor(name string) cpu.ALUBackend {
	if name == "native" {
		return cpu.Native
	}
	return cpu.Gate
}

func runBinary(path string, backend cpu.ALUBackend, seg, off uint16, steps int, trace bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}
	c := cpu.NewCPU8086WithALU(backend)
	c.Seg[cpu.CS], c.IP = seg, off
	c.Seg[cpu.DS] = seg
	c.Seg[cpu.SS] = seg
	c.Seg[cpu.ES] = seg
	c.Regs[cpu.SP] = 0xFFFE
	c.Mem.(*cpu.RAM).LoadAt(cpu.PhysAddr(seg, off), data)

	executed := 0
	for executed < steps && !c.Halted {
		if trace {
			printState(c)
		}
		if err := c.Step(); err != nil {
			fmt.Fprintln(os.Stderr, "stop:", err)
			break
		}
		executed++
	}
	fmt.Printf("eseguite %d istruzioni (halted=%v)\n", executed, c.Halted)
	printState(c)
}

func printState(c *cpu.CPU8086) {
	op := c.Mem.Read8(cpu.PhysAddr(c.Seg[cpu.CS], c.IP))
	fmt.Printf("%04X:%04X op=%02X  AX=%04X BX=%04X CX=%04X DX=%04X  SP=%04X BP=%04X SI=%04X DI=%04X  DS=%04X ES=%04X SS=%04X  F=%04X\n",
		c.Seg[cpu.CS], c.IP, op,
		c.Regs[cpu.AX], c.Regs[cpu.BX], c.Regs[cpu.CX], c.Regs[cpu.DX],
		c.Regs[cpu.SP], c.Regs[cpu.BP], c.Regs[cpu.SI], c.Regs[cpu.DI],
		c.Seg[cpu.DS], c.Seg[cpu.ES], c.Seg[cpu.SS], c.PackFlags())
}

func runSuite(dir string, backend cpu.ALUBackend) {
	res, err := testsuite.RunDir(dir, testsuite.Options{Backend: backend})
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}
	fmt.Printf("SingleStepTests: %d/%d passati (%d falliti)\n", res.Passed, res.Total, res.Failed())
	for i, f := range res.Failures {
		if i >= 30 {
			fmt.Printf("... e altri %d fallimenti\n", res.Failed()-30)
			break
		}
		fmt.Println(" ", f)
	}
	if res.Failed() > 0 {
		os.Exit(1)
	}
}
