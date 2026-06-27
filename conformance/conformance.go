// Package conformance esegue una piccola batteria di programmi auto-verificanti
// sul core 8086, senza dataset esterni. Serve da smoke test rapido (CLI
// -conformance) e gira sempre in CI, a complemento del differenziale Gate/Native
// e della suite SingleStepTests (che richiede invece i vettori esterni).
package conformance

import "github.com/retronet-labs/retronet-8086/cpu"

// Case e' l'esito di un singolo programma di prova.
type Case struct {
	Name   string
	OK     bool
	Detail string
}

// Result riassume l'esito della batteria.
type Result struct {
	Cases []Case
}

// Passed conta i casi superati.
func (r Result) Passed() int {
	n := 0
	for _, c := range r.Cases {
		if c.OK {
			n++
		}
	}
	return n
}

// Failed conta i casi falliti.
func (r Result) Failed() int { return len(r.Cases) - r.Passed() }

type program struct {
	name   string
	code   []byte
	verify func(*cpu.CPU8086) (bool, string)
}

var programs = []program{
	{
		name: "somma 1..5 = 15",
		code: []byte{0xB8, 0, 0, 0xB9, 5, 0, 0x01, 0xC8, 0xE2, 0xFC, 0xF4},
		verify: func(c *cpu.CPU8086) (bool, string) {
			return c.Regs[cpu.AX] == 15, "AX dovrebbe valere 15"
		},
	},
	{
		name: "fattoriale 5! = 120 (MUL a gate)",
		code: []byte{
			0xB8, 1, 0,
			0xBB, 2, 0, 0xF7, 0xE3,
			0xBB, 3, 0, 0xF7, 0xE3,
			0xBB, 4, 0, 0xF7, 0xE3,
			0xBB, 5, 0, 0xF7, 0xE3,
			0xF4,
		},
		verify: func(c *cpu.CPU8086) (bool, string) {
			return c.Regs[cpu.AX] == 120 && c.Regs[cpu.DX] == 0, "AX:DX dovrebbe valere 120:0"
		},
	},
	{
		name: "SHL 1<<4 = 0x10",
		code: []byte{0xB0, 1, 0xB1, 4, 0xD2, 0xE0, 0xF4},
		verify: func(c *cpu.CPU8086) (bool, string) {
			return c.Get8(cpu.AL) == 0x10, "AL dovrebbe valere 0x10"
		},
	},
	{
		name: "IDIV con segno: -10/3 = -3 resto -1",
		code: []byte{
			0xB8, 0xF6, 0xFF, // MOV AX,-10
			0xBB, 0x03, 0x00, // MOV BX,3
			0x99,       // CWD
			0xF7, 0xFB, // IDIV BX
			0xF4,
		},
		verify: func(c *cpu.CPU8086) (bool, string) {
			return c.Regs[cpu.AX] == 0xFFFD && c.Regs[cpu.DX] == 0xFFFF, "AX=-3, DX=-1 attesi"
		},
	},
}

// Run esegue tutti i programmi col backend dato (Gate se nil).
func Run(backend cpu.ALUBackend) Result {
	var res Result
	for _, p := range programs {
		c := cpu.NewCPU8086WithALU(orGate(backend))
		c.Seg[cpu.CS], c.IP = 0, 0x100
		c.Seg[cpu.DS], c.Seg[cpu.SS], c.Seg[cpu.ES] = 0, 0, 0
		c.Regs[cpu.SP] = 0xFFFE
		c.Mem.(*cpu.RAM).LoadAt(cpu.PhysAddr(0, 0x100), p.code)

		if _, err := c.Run(10000); err != nil {
			res.Cases = append(res.Cases, Case{p.name, false, err.Error()})
			continue
		}
		ok, detail := p.verify(c)
		if !ok {
			res.Cases = append(res.Cases, Case{p.name, false, detail})
		} else {
			res.Cases = append(res.Cases, Case{p.name, true, ""})
		}
	}
	return res
}

func orGate(b cpu.ALUBackend) cpu.ALUBackend {
	if b == nil {
		return cpu.Gate
	}
	return b
}
