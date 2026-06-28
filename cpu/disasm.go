package cpu

import "fmt"

// Disassemble decodifica l'istruzione a segment:offset SENZA modificare lo stato
// della CPU, restituendo il testo in sintassi Intel e la sua lunghezza in byte.
// E' simmetrico al decoder di execute.go ed e' usato dal trace della CLI.
func (c *CPU8086) Disassemble(segment, offset uint16) (text string, length int) {
	d := &disasm{c: c, seg: segment, pc: offset, start: offset}
	body := d.decode()
	return d.prefix + body, int(d.pc - d.start)
}

type disasm struct {
	c       *CPU8086
	seg     uint16 // segmento da cui leggere i byte (CS dell'istruzione)
	pc      uint16 // offset corrente
	start   uint16
	prefix  string // rep/lock da anteporre al mnemonico
	segOver string // override di segmento per gli operandi in memoria ("es:" ...)
}

var (
	disReg8  = [8]string{"al", "cl", "dl", "bl", "ah", "ch", "dh", "bh"}
	disReg16 = [8]string{"ax", "cx", "dx", "bx", "sp", "bp", "si", "di"}
	disSeg   = [4]string{"es", "cs", "ss", "ds"}
	disBase  = [8]string{"bx+si", "bx+di", "bp+si", "bp+di", "si", "di", "bp", "bx"}
	disALU   = [8]string{"add", "or", "adc", "sbb", "and", "sub", "xor", "cmp"}
	disShift = [8]string{"rol", "ror", "rcl", "rcr", "shl", "shr", "setmo", "sar"}
	disUnary = [8]string{"test", "test", "not", "neg", "mul", "imul", "div", "idiv"}
	disJcc   = [16]string{"jo", "jno", "jb", "jae", "je", "jne", "jbe", "ja",
		"js", "jns", "jp", "jnp", "jl", "jge", "jle", "jg"}
)

func (d *disasm) u8() byte {
	b := d.c.readMem8(d.seg, d.pc)
	d.pc++
	return b
}

func (d *disasm) u16() uint16 {
	lo := uint16(d.u8())
	hi := uint16(d.u8())
	return lo | hi<<8
}

func (d *disasm) imm8() string  { return fmt.Sprintf("0x%02X", d.u8()) }
func (d *disasm) imm16() string { return fmt.Sprintf("0x%04X", d.u16()) }

func (d *disasm) rel8() string {
	rel := int8(d.u8())
	return fmt.Sprintf("0x%04X", uint16(int32(d.pc)+int32(rel)))
}

func (d *disasm) rel16() string {
	rel := int16(d.u16())
	return fmt.Sprintf("0x%04X", uint16(int32(d.pc)+int32(rel)))
}

// modrm e' la versione "da stampa" di decodeModRM: legge byte ModR/M e
// displacement e produce le stringhe degli operandi, senza toccare lo stato.
type disModRM struct {
	mod, reg, rm byte
	mem          bool
	rmStr        string // gia' formattato (registro o "[...]")
}

func (d *disasm) modrm(w bool) disModRM {
	b := d.u8()
	m := disModRM{mod: b >> 6, reg: b >> 3 & 7, rm: b & 7}
	if m.mod == 3 {
		m.rmStr = d.regName(m.rm, w)
		return m
	}
	m.mem = true
	var expr string
	switch {
	case m.mod == 0 && m.rm == 6:
		expr = fmt.Sprintf("0x%04X", d.u16()) // indirizzo diretto
	default:
		expr = disBase[m.rm]
		switch m.mod {
		case 1:
			disp := int8(d.u8())
			expr += dispStr(int(disp))
		case 2:
			disp := int16(d.u16())
			expr += dispStr(int(disp))
		}
	}
	m.rmStr = "[" + d.segOver + expr + "]"
	return m
}

func dispStr(disp int) string {
	if disp < 0 {
		return fmt.Sprintf("-0x%X", -disp)
	}
	return fmt.Sprintf("+0x%X", disp)
}

func (d *disasm) regName(code byte, w bool) string {
	if w {
		return disReg16[code]
	}
	return disReg8[code]
}

// sized antepone "byte"/"word" agli operandi in memoria quando la dimensione non
// e' implicita da un registro (immediati, gruppi unari, shift).
func sized(m disModRM, w bool) string {
	if !m.mem {
		return m.rmStr
	}
	if w {
		return "word " + m.rmStr
	}
	return "byte " + m.rmStr
}

func (d *disasm) decode() string {
	// Prefissi.
	for {
		op := d.c.readMem8(d.seg, d.pc)
		switch op {
		case 0x26, 0x2E, 0x36, 0x3E:
			d.pc++
			d.segOver = disSeg[(op>>3)&3] + ":"
		case 0xF0:
			d.pc++
			d.prefix += "lock "
		case 0xF2:
			d.pc++
			d.prefix += "repne "
		case 0xF3:
			d.pc++
			d.prefix += "rep "
		default:
			return d.opcode(d.u8())
		}
	}
}

func (d *disasm) opcode(op byte) string {
	// Blocco ALU 0x00-0x3D.
	if op < 0x40 && op&7 < 6 {
		name := disALU[op>>3]
		switch op & 7 {
		case 0:
			m := d.modrm(false)
			return fmt.Sprintf("%s %s, %s", name, m.rmStr, d.regName(m.reg, false))
		case 1:
			m := d.modrm(true)
			return fmt.Sprintf("%s %s, %s", name, m.rmStr, d.regName(m.reg, true))
		case 2:
			m := d.modrm(false)
			return fmt.Sprintf("%s %s, %s", name, d.regName(m.reg, false), m.rmStr)
		case 3:
			m := d.modrm(true)
			return fmt.Sprintf("%s %s, %s", name, d.regName(m.reg, true), m.rmStr)
		case 4:
			return fmt.Sprintf("%s al, %s", name, d.imm8())
		case 5:
			return fmt.Sprintf("%s ax, %s", name, d.imm16())
		}
	}

	switch op {
	case 0x06, 0x0E, 0x16, 0x1E:
		return "push " + disSeg[(op>>3)&3]
	case 0x07, 0x0F, 0x17, 0x1F:
		return "pop " + disSeg[(op>>3)&3]
	case 0x27:
		return "daa"
	case 0x2F:
		return "das"
	case 0x37:
		return "aaa"
	case 0x3F:
		return "aas"

	case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47:
		return "inc " + disReg16[op&7]
	case 0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F:
		return "dec " + disReg16[op&7]
	case 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57:
		return "push " + disReg16[op&7]
	case 0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5E, 0x5F:
		return "pop " + disReg16[op&7]

	case 0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77,
		0x78, 0x79, 0x7A, 0x7B, 0x7C, 0x7D, 0x7E, 0x7F,
		0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67,
		0x68, 0x69, 0x6A, 0x6B, 0x6C, 0x6D, 0x6E, 0x6F:
		return disJcc[op&0x0F] + " " + d.rel8()

	case 0x80, 0x81, 0x82, 0x83:
		w := op&1 == 1
		m := d.modrm(w)
		var imm string
		switch op {
		case 0x81:
			imm = d.imm16()
		default:
			imm = d.imm8()
		}
		return fmt.Sprintf("%s %s, %s", disALU[m.reg], sized(m, w), imm)
	case 0x84, 0x85:
		w := op&1 == 1
		m := d.modrm(w)
		return fmt.Sprintf("test %s, %s", m.rmStr, d.regName(m.reg, w))
	case 0x86, 0x87:
		w := op&1 == 1
		m := d.modrm(w)
		return fmt.Sprintf("xchg %s, %s", m.rmStr, d.regName(m.reg, w))
	case 0x88, 0x89:
		w := op&1 == 1
		m := d.modrm(w)
		return fmt.Sprintf("mov %s, %s", m.rmStr, d.regName(m.reg, w))
	case 0x8A, 0x8B:
		w := op&1 == 1
		m := d.modrm(w)
		return fmt.Sprintf("mov %s, %s", d.regName(m.reg, w), m.rmStr)
	case 0x8C:
		m := d.modrm(true)
		return fmt.Sprintf("mov %s, %s", m.rmStr, disSeg[m.reg&3])
	case 0x8D:
		m := d.modrm(true)
		return fmt.Sprintf("lea %s, %s", disReg16[m.reg], m.rmStr)
	case 0x8E:
		m := d.modrm(true)
		return fmt.Sprintf("mov %s, %s", disSeg[m.reg&3], m.rmStr)
	case 0x8F:
		m := d.modrm(true)
		return "pop " + sized(m, true)

	case 0x90:
		return "nop"
	case 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97:
		return "xchg ax, " + disReg16[op&7]
	case 0x98:
		return "cbw"
	case 0x99:
		return "cwd"
	case 0x9A:
		off := d.imm16()
		seg := d.imm16()
		return fmt.Sprintf("call %s:%s", seg, off)
	case 0x9B:
		return "wait"
	case 0x9C:
		return "pushf"
	case 0x9D:
		return "popf"
	case 0x9E:
		return "sahf"
	case 0x9F:
		return "lahf"

	case 0xA0:
		return fmt.Sprintf("mov al, [%s%s]", d.segOver, d.imm16())
	case 0xA1:
		return fmt.Sprintf("mov ax, [%s%s]", d.segOver, d.imm16())
	case 0xA2:
		return fmt.Sprintf("mov [%s%s], al", d.segOver, d.imm16())
	case 0xA3:
		return fmt.Sprintf("mov [%s%s], ax", d.segOver, d.imm16())
	case 0xA4:
		return "movsb"
	case 0xA5:
		return "movsw"
	case 0xA6:
		return "cmpsb"
	case 0xA7:
		return "cmpsw"
	case 0xA8:
		return "test al, " + d.imm8()
	case 0xA9:
		return "test ax, " + d.imm16()
	case 0xAA:
		return "stosb"
	case 0xAB:
		return "stosw"
	case 0xAC:
		return "lodsb"
	case 0xAD:
		return "lodsw"
	case 0xAE:
		return "scasb"
	case 0xAF:
		return "scasw"

	case 0xB0, 0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7:
		return fmt.Sprintf("mov %s, %s", disReg8[op&7], d.imm8())
	case 0xB8, 0xB9, 0xBA, 0xBB, 0xBC, 0xBD, 0xBE, 0xBF:
		return fmt.Sprintf("mov %s, %s", disReg16[op&7], d.imm16())

	case 0xC0, 0xC2:
		return "ret " + d.imm16()
	case 0xC1, 0xC3:
		return "ret"
	case 0xC4:
		m := d.modrm(true)
		return fmt.Sprintf("les %s, %s", disReg16[m.reg], m.rmStr)
	case 0xC5:
		m := d.modrm(true)
		return fmt.Sprintf("lds %s, %s", disReg16[m.reg], m.rmStr)
	case 0xC6:
		m := d.modrm(false)
		return fmt.Sprintf("mov %s, %s", sized(m, false), d.imm8())
	case 0xC7:
		m := d.modrm(true)
		return fmt.Sprintf("mov %s, %s", sized(m, true), d.imm16())
	case 0xC8, 0xCA:
		return "retf " + d.imm16()
	case 0xC9, 0xCB:
		return "retf"
	case 0xCC:
		return "int3"
	case 0xCD:
		return "int " + d.imm8()
	case 0xCE:
		return "into"
	case 0xCF:
		return "iret"

	case 0xD0, 0xD1:
		w := op&1 == 1
		m := d.modrm(w)
		return fmt.Sprintf("%s %s, 1", disShift[m.reg], sized(m, w))
	case 0xD2, 0xD3:
		w := op&1 == 1
		m := d.modrm(w)
		return fmt.Sprintf("%s %s, cl", disShift[m.reg], sized(m, w))
	case 0xD4:
		return "aam " + d.imm8()
	case 0xD5:
		return "aad " + d.imm8()
	case 0xD6:
		return "salc"
	case 0xD7:
		return "xlat"
	case 0xD8, 0xD9, 0xDA, 0xDB, 0xDC, 0xDD, 0xDE, 0xDF:
		m := d.modrm(true)
		return "esc " + m.rmStr

	case 0xE0:
		return "loopne " + d.rel8()
	case 0xE1:
		return "loope " + d.rel8()
	case 0xE2:
		return "loop " + d.rel8()
	case 0xE3:
		return "jcxz " + d.rel8()
	case 0xE4:
		return "in al, " + d.imm8()
	case 0xE5:
		return "in ax, " + d.imm8()
	case 0xE6:
		return "out " + d.imm8() + ", al"
	case 0xE7:
		return "out " + d.imm8() + ", ax"
	case 0xE8:
		return "call " + d.rel16()
	case 0xE9:
		return "jmp " + d.rel16()
	case 0xEA:
		off := d.imm16()
		seg := d.imm16()
		return fmt.Sprintf("jmp %s:%s", seg, off)
	case 0xEB:
		return "jmp " + d.rel8()
	case 0xEC:
		return "in al, dx"
	case 0xED:
		return "in ax, dx"
	case 0xEE:
		return "out dx, al"
	case 0xEF:
		return "out dx, ax"

	case 0xF4:
		return "hlt"
	case 0xF5:
		return "cmc"
	case 0xF6, 0xF7:
		w := op&1 == 1
		m := d.modrm(w)
		switch m.reg {
		case 0, 1:
			if w {
				return fmt.Sprintf("test %s, %s", sized(m, w), d.imm16())
			}
			return fmt.Sprintf("test %s, %s", sized(m, w), d.imm8())
		default:
			return fmt.Sprintf("%s %s", disUnary[m.reg], sized(m, w))
		}
	case 0xF8:
		return "clc"
	case 0xF9:
		return "stc"
	case 0xFA:
		return "cli"
	case 0xFB:
		return "sti"
	case 0xFC:
		return "cld"
	case 0xFD:
		return "std"
	case 0xFE:
		m := d.modrm(false)
		if m.reg == 1 {
			return "dec " + sized(m, false)
		}
		return "inc " + sized(m, false)
	case 0xFF:
		m := d.modrm(true)
		switch m.reg {
		case 0:
			return "inc " + sized(m, true)
		case 1:
			return "dec " + sized(m, true)
		case 2:
			return "call " + m.rmStr
		case 3:
			return "call far " + m.rmStr
		case 4:
			return "jmp " + m.rmStr
		case 5:
			return "jmp far " + m.rmStr
		default: // 6, 7
			return "push " + sized(m, true)
		}
	}

	return fmt.Sprintf("db 0x%02X", op)
}
