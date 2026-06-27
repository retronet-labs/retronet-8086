package cpu

import "github.com/retronet-labs/retronet-hardware/bridge/i8086"

// NewCPU8086 crea una CPU pronta all'uso: profilo 8088 (default coerente con
// l'IBM PC/XT), ALU a porte logiche, RAM da 1 MB e spazio I/O no-op. Lo stato dei
// registri e' quello di reset dell'8086.
func NewCPU8086() *CPU8086 {
	c := &CPU8086{
		Profile: Profile8088,
		Mem:     NewRAM(),
		IO:      nullPorts{},
		alu:     Gate,
	}
	c.Reset()
	return c
}

// NewCPU8086WithALU crea una CPU scegliendo il backend aritmetico-logico.
func NewCPU8086WithALU(backend ALUBackend) *CPU8086 {
	c := NewCPU8086()
	c.SetALU(backend)
	return c
}

// Reset riporta i registri allo stato di accensione dell'8086: CS=0xFFFF, IP=0
// (quindi il primo fetch e' a 0xFFFF0), gli altri segmenti e i generali a 0, i
// flag ai soli bit riservati. Memoria e I/O non vengono toccati.
func (c *CPU8086) Reset() {
	c.Regs = [8]uint16{}
	c.Seg = [4]uint16{}
	c.Seg[CS] = 0xFFFF
	c.IP = 0x0000
	c.SetFlags(flagsReserved)
	c.Halted = false
}

// --- Accesso alla memoria (segmento:offset, con wrap a 16 bit dell'offset) ---

func (c *CPU8086) readMem8(segment, offset uint16) byte {
	return c.Mem.Read8(PhysAddr(segment, offset))
}

func (c *CPU8086) writeMem8(segment, offset uint16, value byte) {
	c.Mem.Write8(PhysAddr(segment, offset), value)
}

// readMem16 legge una parola little-endian; il secondo byte usa offset+1 con wrap
// a 16 bit nello stesso segmento, come sull'8086.
func (c *CPU8086) readMem16(segment, offset uint16) uint16 {
	lo := uint16(c.readMem8(segment, offset))
	hi := uint16(c.readMem8(segment, offset+1))
	return lo | hi<<8
}

func (c *CPU8086) writeMem16(segment, offset uint16, value uint16) {
	c.writeMem8(segment, offset, byte(value))
	c.writeMem8(segment, offset+1, byte(value>>8))
}

// --- Fetch del flusso d'istruzioni da CS:IP ---

func (c *CPU8086) fetch8() byte {
	b := c.readMem8(c.Seg[CS], c.IP)
	c.IP++
	return b
}

func (c *CPU8086) fetch16() uint16 {
	lo := uint16(c.fetch8())
	hi := uint16(c.fetch8())
	return lo | hi<<8
}

// --- Stack su SS:SP (cresce verso il basso) ---

func (c *CPU8086) push16(value uint16) {
	c.Regs[SP] -= 2
	c.writeMem16(c.Seg[SS], c.Regs[SP], value)
}

func (c *CPU8086) pop16() uint16 {
	value := c.readMem16(c.Seg[SS], c.Regs[SP])
	c.Regs[SP] += 2
	return value
}

// --- Applicazione dei flag prodotti dalla ALU ---

// applyArithFlags copia tutti i flag aritmetici (incluso il Carry) nei campi CPU.
func (c *CPU8086) applyArithFlags(f i8086.Flags) {
	c.CF = f.Carry
	c.PF = f.Parity
	c.AF = f.Auxiliary
	c.ZF = f.Zero
	c.SF = f.Sign
	c.OF = f.Overflow
}

// applyIncDecFlags copia i flag di INC/DEC: tutti tranne il Carry, che l'8086 non
// modifica per queste istruzioni.
func (c *CPU8086) applyIncDecFlags(f i8086.Flags) {
	c.PF = f.Parity
	c.AF = f.Auxiliary
	c.ZF = f.Zero
	c.SF = f.Sign
	c.OF = f.Overflow
}
