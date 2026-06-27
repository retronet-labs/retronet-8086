package cpu

// CPU8086 e' lo stato del processore in real mode. I registri generali sono
// indicizzati da Reg16; i mezzi-registri da Reg8 condividono lo stesso storage.
// I flag sono campi booleani distinti (come negli altri core RetroNet) e si
// impacchettano nel registro FLAGS a 16 bit solo quando serve (PUSHF, IRET,
// confronto con i vettori di test).
type CPU8086 struct {
	Regs [8]uint16 // generali, indicizzati da Reg16
	Seg  [4]uint16 // segmenti, indicizzati da Sreg
	IP   uint16

	// Flag aritmetici (prodotti dalla ALU) e di controllo (gestiti dalla CPU).
	CF, PF, AF, ZF, SF, TF, IF, DF, OF bool

	Halted  bool
	Profile Profile

	Mem Bus   // spazio di memoria fisico (default: RAM da 1 MB)
	IO  Ports // spazio I/O a 64 K porte (default: no-op)

	alu ALUBackend // motore aritmetico-logico (default: Gate)
}

// Get16 legge un registro generale a 16 bit.
func (c *CPU8086) Get16(r Reg16) uint16 { return c.Regs[r] }

// Set16 scrive un registro generale a 16 bit.
func (c *CPU8086) Set16(r Reg16, v uint16) { c.Regs[r] = v }

// Get8 legge un mezzo-registro a 8 bit. I codici 0-3 sono le meta' basse
// (AL/CL/DL/BL), i codici 4-7 le meta' alte (AH/CH/DH/BH) degli stessi AX/CX/DX/BX.
func (c *CPU8086) Get8(r Reg8) byte {
	full := c.Regs[r&3]
	if r < 4 {
		return byte(full)
	}
	return byte(full >> 8)
}

// Set8 scrive un mezzo-registro a 8 bit lasciando intatta l'altra meta'.
func (c *CPU8086) Set8(r Reg8, v byte) {
	idx := r & 3
	full := c.Regs[idx]
	if r < 4 {
		c.Regs[idx] = full&0xFF00 | uint16(v)
	} else {
		c.Regs[idx] = full&0x00FF | uint16(v)<<8
	}
}

// GetSeg legge un registro di segmento.
func (c *CPU8086) GetSeg(s Sreg) uint16 { return c.Seg[s] }

// SetSeg scrive un registro di segmento.
func (c *CPU8086) SetSeg(s Sreg, v uint16) { c.Seg[s] = v }

// PhysAddr calcola l'indirizzo fisico a 20 bit dalla coppia segmento:offset
// secondo la formula dell'8086: (segment << 4) + offset, con wrap a 1 MB.
func PhysAddr(segment, offset uint16) uint32 {
	return (uint32(segment)<<4 + uint32(offset)) & AddressMask
}
