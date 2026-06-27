package cpu

// prefixes raccoglie i prefissi d'istruzione raccolti prima dell'opcode.
type prefixes struct {
	segOverride Sreg // segmento da usare al posto del default
	hasSeg      bool
	rep         byte // 0 = nessuno, 0xF2 = REPNE/REPNZ, 0xF3 = REP/REPE/REPZ
	lock        bool
}

// modrm e' un byte ModR/M decodificato. Per un operando in memoria, seg e off
// contengono gia' segmento (valore, override applicato) e offset effettivi;
// l'eventuale displacement e' stato letto dal flusso d'istruzioni.
type modrm struct {
	mod byte
	reg byte // campo reg: registro oppure estensione di opcode nei gruppi
	rm  byte // campo rm: indice di registro se !mem
	mem bool
	seg uint16
	off uint16
}

// decodeModRM legge il byte ModR/M (ed eventuale displacement) e ne calcola
// l'operando, applicando l'eventuale override di segmento.
func (c *CPU8086) decodeModRM(pfx prefixes) modrm {
	b := c.fetch8()
	m := modrm{
		mod: b >> 6,
		reg: b >> 3 & 0x07,
		rm:  b & 0x07,
	}
	if m.mod == 0x03 {
		return m // rm e' un registro
	}
	m.mem = true
	m.seg, m.off = c.effectiveAddr(m.mod, m.rm, pfx)
	return m
}

// effectiveAddr calcola la coppia segmento:offset dell'operando in memoria
// secondo le otto combinazioni base/indice dell'8086. Il default e' DS, tranne
// quando l'indirizzamento usa BP (allora SS); un prefisso di override sostituisce
// il segmento.
func (c *CPU8086) effectiveAddr(mod, rm byte, pfx prefixes) (uint16, uint16) {
	var base uint16
	defSeg := DS
	switch rm {
	case 0:
		base = c.Regs[BX] + c.Regs[SI]
	case 1:
		base = c.Regs[BX] + c.Regs[DI]
	case 2:
		base = c.Regs[BP] + c.Regs[SI]
		defSeg = SS
	case 3:
		base = c.Regs[BP] + c.Regs[DI]
		defSeg = SS
	case 4:
		base = c.Regs[SI]
	case 5:
		base = c.Regs[DI]
	case 6:
		if mod == 0 {
			base = c.fetch16() // indirizzo diretto disp16
		} else {
			base = c.Regs[BP]
			defSeg = SS
		}
	case 7:
		base = c.Regs[BX]
	}

	var disp uint16
	switch mod {
	case 1:
		disp = uint16(int16(int8(c.fetch8()))) // disp8 con estensione di segno
	case 2:
		disp = c.fetch16()
	}

	seg := defSeg
	if pfx.hasSeg {
		seg = pfx.segOverride
	}
	return c.Seg[seg], base + disp
}

// width restituisce la larghezza in bit a partire dal bit w dell'opcode.
func width(w bool) int {
	if w {
		return 16
	}
	return 8
}

// rmRead legge l'operando rm alla larghezza data (w=true -> 16 bit).
func (c *CPU8086) rmRead(m modrm, w bool) uint16 {
	if !m.mem {
		return c.regValue(m.rm, w)
	}
	if w {
		return c.readMem16(m.seg, m.off)
	}
	return uint16(c.readMem8(m.seg, m.off))
}

// rmWrite scrive l'operando rm alla larghezza data.
func (c *CPU8086) rmWrite(m modrm, w bool, v uint16) {
	if !m.mem {
		c.setRegValue(m.rm, w, v)
		return
	}
	if w {
		c.writeMem16(m.seg, m.off, v)
		return
	}
	c.writeMem8(m.seg, m.off, byte(v))
}

// regValue legge il registro indicizzato da code alla larghezza data, usando la
// mappa a 8 bit (AL..BH) oppure a 16 bit (AX..DI).
func (c *CPU8086) regValue(code byte, w bool) uint16 {
	if w {
		return c.Regs[code]
	}
	return uint16(c.Get8(Reg8(code)))
}

func (c *CPU8086) setRegValue(code byte, w bool, v uint16) {
	if w {
		c.Regs[code] = v
		return
	}
	c.Set8(Reg8(code), byte(v))
}
