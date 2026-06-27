package cpu

import "github.com/retronet-labs/retronet-hardware/bridge/i8086"

// execute decodifica ed esegue un opcode (i prefissi sono gia' stati raccolti).
func (c *CPU8086) execute(op byte, pfx prefixes) error {
	// Blocco aritmetico-logico 0x00-0x3D: per ogni gruppo (ADD..CMP) le sei forme
	// r/m,r | r,r/m | acc,imm a 8 e 16 bit.
	if op < 0x40 && op&0x07 < 6 {
		group := op >> 3
		switch op & 0x07 {
		case 0:
			c.aluRMReg(group, false, false, pfx)
		case 1:
			c.aluRMReg(group, true, false, pfx)
		case 2:
			c.aluRMReg(group, false, true, pfx)
		case 3:
			c.aluRMReg(group, true, true, pfx)
		case 4:
			c.aluAccImm(group, false)
		case 5:
			c.aluAccImm(group, true)
		}
		return nil
	}

	switch op {
	// --- MOV registro/memoria ---
	case 0x88:
		c.movRMReg(false, false, pfx)
	case 0x89:
		c.movRMReg(true, false, pfx)
	case 0x8A:
		c.movRMReg(false, true, pfx)
	case 0x8B:
		c.movRMReg(true, true, pfx)
	case 0x8C: // MOV r/m16, sreg
		m := c.decodeModRM(pfx)
		c.rmWrite(m, true, c.Seg[Sreg(m.reg&0x03)])
	case 0x8E: // MOV sreg, r/m16
		m := c.decodeModRM(pfx)
		c.Seg[Sreg(m.reg&0x03)] = c.rmRead(m, true)
	case 0x8D: // LEA r16, m
		m := c.decodeModRM(pfx)
		c.Regs[m.reg] = m.off
	case 0xA0: // MOV AL, [moffs]
		c.Set8(AL, c.readMem8(c.segFor(DS, pfx), c.fetch16()))
	case 0xA1: // MOV AX, [moffs]
		c.Regs[AX] = c.readMem16(c.segFor(DS, pfx), c.fetch16())
	case 0xA2: // MOV [moffs], AL
		c.writeMem8(c.segFor(DS, pfx), c.fetch16(), c.Get8(AL))
	case 0xA3: // MOV [moffs], AX
		c.writeMem16(c.segFor(DS, pfx), c.fetch16(), c.Regs[AX])
	case 0xC6: // MOV r/m8, imm8
		m := c.decodeModRM(pfx)
		c.rmWrite(m, false, uint16(c.fetch8()))
	case 0xC7: // MOV r/m16, imm16
		m := c.decodeModRM(pfx)
		c.rmWrite(m, true, c.fetch16())

	// --- MOV reg, imm ---
	case 0xB0, 0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7:
		c.Set8(Reg8(op&0x07), c.fetch8())
	case 0xB8, 0xB9, 0xBA, 0xBB, 0xBC, 0xBD, 0xBE, 0xBF:
		c.Regs[op&0x07] = c.fetch16()

	// --- XCHG ---
	case 0x86:
		c.xchgRMReg(false, pfx)
	case 0x87:
		c.xchgRMReg(true, pfx)
	case 0x90: // NOP = XCHG AX,AX
	case 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97:
		r := op & 0x07
		c.Regs[AX], c.Regs[r] = c.Regs[r], c.Regs[AX]

	// --- PUSH/POP ---
	case 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57:
		c.push16(c.Regs[op&0x07])
	case 0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5E, 0x5F:
		c.Regs[op&0x07] = c.pop16()
	case 0x06:
		c.push16(c.Seg[ES])
	case 0x0E:
		c.push16(c.Seg[CS])
	case 0x16:
		c.push16(c.Seg[SS])
	case 0x1E:
		c.push16(c.Seg[DS])
	case 0x07:
		c.Seg[ES] = c.pop16()
	case 0x17:
		c.Seg[SS] = c.pop16()
	case 0x1F:
		c.Seg[DS] = c.pop16()
	case 0x8F: // POP r/m16
		m := c.decodeModRM(pfx)
		c.rmWrite(m, true, c.pop16())
	case 0x9C: // PUSHF
		c.push16(c.PackFlags())
	case 0x9D: // POPF
		c.SetFlags(c.pop16())

	// --- TEST ---
	case 0x84:
		c.testRMReg(false, pfx)
	case 0x85:
		c.testRMReg(true, pfx)
	case 0xA8: // TEST AL, imm8
		_, f := c.backend().ALU(i8086.GroupAND, uint16(c.Get8(AL)), uint16(c.fetch8()), 8, false)
		c.applyArithFlags(f)
	case 0xA9: // TEST AX, imm16
		_, f := c.backend().ALU(i8086.GroupAND, c.Regs[AX], c.fetch16(), 16, false)
		c.applyArithFlags(f)

	// --- gruppi a opcode esteso ---
	case 0x80, 0x81, 0x82, 0x83:
		c.aluImmGroup(op, pfx)
	case 0xF6:
		return c.unaryGroup(false, pfx)
	case 0xF7:
		return c.unaryGroup(true, pfx)
	case 0xFE:
		c.incDecRM(false, pfx)
	case 0xFF:
		c.group0xFF(pfx)

	// --- flag e conversioni ---
	case 0xF5:
		c.CF = !c.CF
	case 0xF8:
		c.CF = false
	case 0xF9:
		c.CF = true
	case 0xFA:
		c.IF = false
	case 0xFB:
		c.IF = true
	case 0xFC:
		c.DF = false
	case 0xFD:
		c.DF = true
	case 0x9E: // SAHF
		ah := c.Get8(AH)
		c.CF = ah&FlagCF != 0
		c.PF = ah&FlagPF != 0
		c.AF = ah&FlagAF != 0
		c.ZF = ah&FlagZF != 0
		c.SF = ah&0x80 != 0
	case 0x9F: // LAHF
		c.Set8(AH, byte(c.PackFlags()))
	case 0x98: // CBW
		if c.Get8(AL)&0x80 != 0 {
			c.Set8(AH, 0xFF)
		} else {
			c.Set8(AH, 0x00)
		}
	case 0x99: // CWD
		if c.Regs[AX]&0x8000 != 0 {
			c.Regs[DX] = 0xFFFF
		} else {
			c.Regs[DX] = 0x0000
		}

	// --- INC/DEC reg16 ---
	case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47:
		res, f := c.backend().Increment(c.Regs[op&0x07], 16)
		c.applyIncDecFlags(f)
		c.Regs[op&0x07] = res
	case 0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F:
		res, f := c.backend().Decrement(c.Regs[op&0x07], 16)
		c.applyIncDecFlags(f)
		c.Regs[op&0x07] = res

	// --- salti, chiamate, ritorni ---
	case 0xEB: // JMP rel8
		rel := int8(c.fetch8())
		c.IP = uint16(int32(c.IP) + int32(rel))
	case 0xE9: // JMP rel16
		rel := int16(c.fetch16())
		c.IP = uint16(int32(c.IP) + int32(rel))
	case 0xEA: // JMP far ptr16:16
		off := c.fetch16()
		seg := c.fetch16()
		c.IP, c.Seg[CS] = off, seg
	case 0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77,
		0x78, 0x79, 0x7A, 0x7B, 0x7C, 0x7D, 0x7E, 0x7F:
		rel := int8(c.fetch8())
		if c.condition(op & 0x0F) {
			c.IP = uint16(int32(c.IP) + int32(rel))
		}
	case 0xE0, 0xE1, 0xE2: // LOOPNE / LOOPE / LOOP
		rel := int8(c.fetch8())
		c.Regs[CX]--
		take := c.Regs[CX] != 0
		switch op {
		case 0xE0:
			take = take && !c.ZF
		case 0xE1:
			take = take && c.ZF
		}
		if take {
			c.IP = uint16(int32(c.IP) + int32(rel))
		}
	case 0xE3: // JCXZ rel8
		rel := int8(c.fetch8())
		if c.Regs[CX] == 0 {
			c.IP = uint16(int32(c.IP) + int32(rel))
		}
	case 0xE8: // CALL rel16
		rel := int16(c.fetch16())
		c.push16(c.IP)
		c.IP = uint16(int32(c.IP) + int32(rel))
	case 0xC3: // RET
		c.IP = c.pop16()
	case 0xC2: // RET imm16
		n := c.fetch16()
		c.IP = c.pop16()
		c.Regs[SP] += n
	case 0xCB: // RETF
		c.IP = c.pop16()
		c.Seg[CS] = c.pop16()
	case 0xCA: // RETF imm16
		n := c.fetch16()
		c.IP = c.pop16()
		c.Seg[CS] = c.pop16()
		c.Regs[SP] += n

	// --- interrupt ---
	case 0xCC: // INT3
		c.raiseInterrupt(3)
	case 0xCD: // INT imm8
		c.raiseInterrupt(c.fetch8())
	case 0xCE: // INTO
		if c.OF {
			c.raiseInterrupt(4)
		}
	case 0xCF: // IRET
		c.iret()

	// --- I/O ---
	case 0xE4:
		c.Set8(AL, c.IO.In8(uint16(c.fetch8())))
	case 0xE5:
		c.Regs[AX] = c.in16(uint16(c.fetch8()))
	case 0xE6:
		c.IO.Out8(uint16(c.fetch8()), c.Get8(AL))
	case 0xE7:
		c.out16(uint16(c.fetch8()), c.Regs[AX])
	case 0xEC:
		c.Set8(AL, c.IO.In8(c.Regs[DX]))
	case 0xED:
		c.Regs[AX] = c.in16(c.Regs[DX])
	case 0xEE:
		c.IO.Out8(c.Regs[DX], c.Get8(AL))
	case 0xEF:
		c.out16(c.Regs[DX], c.Regs[AX])

	// --- controllo ---
	case 0xF4: // HLT
		c.Halted = true

	default:
		return &UnimplementedError{Opcode: op, CS: c.Seg[CS], IP: c.IP}
	}
	return nil
}

// --- helper di esecuzione ---

func (c *CPU8086) aluRMReg(group byte, w, dir bool, pfx prefixes) {
	m := c.decodeModRM(pfx)
	rmv := c.rmRead(m, w)
	rv := c.regValue(m.reg, w)
	a, b := rmv, rv
	if dir {
		a, b = rv, rmv
	}
	res, f := c.backend().ALU(group, a, b, width(w), c.CF)
	c.applyArithFlags(f)
	if group != i8086.GroupCMP {
		if dir {
			c.setRegValue(m.reg, w, res)
		} else {
			c.rmWrite(m, w, res)
		}
	}
}

func (c *CPU8086) aluAccImm(group byte, w bool) {
	b := c.immediate(w)
	a := c.regValue(0, w) // AL / AX
	res, f := c.backend().ALU(group, a, b, width(w), c.CF)
	c.applyArithFlags(f)
	if group != i8086.GroupCMP {
		c.setRegValue(0, w, res)
	}
}

func (c *CPU8086) aluImmGroup(op byte, pfx prefixes) {
	w := op&0x01 == 1
	m := c.decodeModRM(pfx)
	group := m.reg
	a := c.rmRead(m, w)
	var b uint16
	switch op {
	case 0x81:
		b = c.fetch16()
	case 0x83:
		b = uint16(int16(int8(c.fetch8()))) // imm8 con estensione di segno
	default: // 0x80, 0x82
		b = uint16(c.fetch8())
	}
	res, f := c.backend().ALU(group, a, b, width(w), c.CF)
	c.applyArithFlags(f)
	if group != i8086.GroupCMP {
		c.rmWrite(m, w, res)
	}
}

func (c *CPU8086) movRMReg(w, toReg bool, pfx prefixes) {
	m := c.decodeModRM(pfx)
	if toReg {
		c.setRegValue(m.reg, w, c.rmRead(m, w))
	} else {
		c.rmWrite(m, w, c.regValue(m.reg, w))
	}
}

func (c *CPU8086) xchgRMReg(w bool, pfx prefixes) {
	m := c.decodeModRM(pfx)
	rmv := c.rmRead(m, w)
	rv := c.regValue(m.reg, w)
	c.rmWrite(m, w, rv)
	c.setRegValue(m.reg, w, rmv)
}

func (c *CPU8086) testRMReg(w bool, pfx prefixes) {
	m := c.decodeModRM(pfx)
	_, f := c.backend().ALU(i8086.GroupAND, c.rmRead(m, w), c.regValue(m.reg, w), width(w), false)
	c.applyArithFlags(f)
}

func (c *CPU8086) unaryGroup(w bool, pfx prefixes) error {
	m := c.decodeModRM(pfx)
	switch m.reg {
	case 0, 1: // TEST r/m, imm
		a := c.rmRead(m, w)
		b := c.immediate(w)
		_, f := c.backend().ALU(i8086.GroupAND, a, b, width(w), false)
		c.applyArithFlags(f)
	case 2: // NOT (non tocca i flag)
		c.rmWrite(m, w, (^c.rmRead(m, w))&maskW(w))
	case 3: // NEG = 0 - x
		res, f := c.backend().ALU(i8086.GroupSUB, 0, c.rmRead(m, w), width(w), false)
		c.applyArithFlags(f)
		c.rmWrite(m, w, res)
	case 4:
		c.mul(m, w, false)
	case 5:
		c.mul(m, w, true)
	case 6:
		c.div(m, w, false)
	case 7:
		c.div(m, w, true)
	}
	return nil
}

func (c *CPU8086) mul(m modrm, w, signed bool) {
	src := c.rmRead(m, w)
	if w {
		prod, of := c.backend().Mul(c.Regs[AX], src, 16, signed)
		c.Regs[AX] = uint16(prod)
		c.Regs[DX] = uint16(prod >> 16)
		c.CF, c.OF = of, of
	} else {
		prod, of := c.backend().Mul(uint16(c.Get8(AL)), src, 8, signed)
		c.Regs[AX] = uint16(prod)
		c.CF, c.OF = of, of
	}
}

func (c *CPU8086) div(m modrm, w, signed bool) {
	src := c.rmRead(m, w)
	if w {
		dividend := uint32(c.Regs[DX])<<16 | uint32(c.Regs[AX])
		q, r, ok := c.backend().Div(dividend, src, 16, signed)
		if !ok {
			c.raiseInterrupt(0) // #DE
			return
		}
		c.Regs[AX], c.Regs[DX] = q, r
	} else {
		q, r, ok := c.backend().Div(uint32(c.Regs[AX]), src, 8, signed)
		if !ok {
			c.raiseInterrupt(0)
			return
		}
		c.Set8(AL, byte(q))
		c.Set8(AH, byte(r))
	}
}

func (c *CPU8086) incDecRM(w bool, pfx prefixes) {
	m := c.decodeModRM(pfx)
	v := c.rmRead(m, w)
	var res uint16
	var f i8086.Flags
	if m.reg == 0 {
		res, f = c.backend().Increment(v, width(w))
	} else {
		res, f = c.backend().Decrement(v, width(w))
	}
	c.applyIncDecFlags(f)
	c.rmWrite(m, w, res)
}

func (c *CPU8086) group0xFF(pfx prefixes) {
	m := c.decodeModRM(pfx)
	switch m.reg {
	case 0: // INC r/m16
		res, f := c.backend().Increment(c.rmRead(m, true), 16)
		c.applyIncDecFlags(f)
		c.rmWrite(m, true, res)
	case 1: // DEC r/m16
		res, f := c.backend().Decrement(c.rmRead(m, true), 16)
		c.applyIncDecFlags(f)
		c.rmWrite(m, true, res)
	case 2: // CALL r/m16 (near indiretto)
		target := c.rmRead(m, true)
		c.push16(c.IP)
		c.IP = target
	case 3: // CALL m16:16 (far indiretto)
		off := c.readMem16(m.seg, m.off)
		seg := c.readMem16(m.seg, m.off+2)
		c.push16(c.Seg[CS])
		c.push16(c.IP)
		c.IP, c.Seg[CS] = off, seg
	case 4: // JMP r/m16
		c.IP = c.rmRead(m, true)
	case 5: // JMP m16:16 (far indiretto)
		c.IP = c.readMem16(m.seg, m.off)
		c.Seg[CS] = c.readMem16(m.seg, m.off+2)
	case 6: // PUSH r/m16
		c.push16(c.rmRead(m, true))
	}
}

// condition valuta la condizione a 4 bit dei salti condizionati (0x70-0x7F).
func (c *CPU8086) condition(code byte) bool {
	switch code {
	case 0x0:
		return c.OF
	case 0x1:
		return !c.OF
	case 0x2:
		return c.CF
	case 0x3:
		return !c.CF
	case 0x4:
		return c.ZF
	case 0x5:
		return !c.ZF
	case 0x6:
		return c.CF || c.ZF
	case 0x7:
		return !c.CF && !c.ZF
	case 0x8:
		return c.SF
	case 0x9:
		return !c.SF
	case 0xA:
		return c.PF
	case 0xB:
		return !c.PF
	case 0xC:
		return c.SF != c.OF
	case 0xD:
		return c.SF == c.OF
	case 0xE:
		return c.ZF || (c.SF != c.OF)
	default: // 0xF
		return !c.ZF && (c.SF == c.OF)
	}
}

func (c *CPU8086) immediate(w bool) uint16 {
	if w {
		return c.fetch16()
	}
	return uint16(c.fetch8())
}

func (c *CPU8086) segFor(def Sreg, pfx prefixes) uint16 {
	if pfx.hasSeg {
		return c.Seg[pfx.segOverride]
	}
	return c.Seg[def]
}

func maskW(w bool) uint16 {
	if w {
		return 0xFFFF
	}
	return 0xFF
}
