package cpu

import (
	"math/bits"

	"github.com/retronet-labs/retronet-hardware/bridge/i8086"
)

// nativeBackend riproduce la semantica aritmetico-logica dell'8086 con i normali
// operatori di Go. E' funzionalmente identico al backend a porte (Gate) ma piu'
// rapido: utile come oracolo del test differenziale.
type nativeBackend struct{}

func widthMask(width int) uint32 { return uint32(1)<<uint(width) - 1 }

func parityEvenLow8(v uint32) bool { return bits.OnesCount8(byte(v))%2 == 0 }

func signExtend(v uint32, width int) int64 {
	if v>>uint(width-1)&1 == 1 {
		return int64(v) - int64(uint64(1)<<uint(width))
	}
	return int64(v)
}

// natArith calcola a + addend + cin (per la sottrazione addend e' il complemento
// di b entro width e isSub inverte il Carry come prestito), derivando i flag dai
// riporti.
func natArith(a, b uint16, width int, cin, isSub bool) (uint16, i8086.Flags) {
	mask := widthMask(width)
	av := uint32(a) & mask
	bv := uint32(b) & mask
	addend := bv
	if isSub {
		addend = (^bv) & mask
	}
	c := uint32(0)
	if cin {
		c = 1
	}
	res := av + addend + c
	out := res & mask
	xorc := res ^ av ^ addend
	carryOut := res>>uint(width)&1 == 1
	carryMSB := xorc>>uint(width-1)&1 == 1
	af := (av^bv^out)>>4&1 == 1 // AF dai valori originali (b non complementato)

	carry := carryOut
	if isSub {
		carry = !carryOut
	}
	return uint16(out), i8086.Flags{
		Carry:     carry,
		Parity:    parityEvenLow8(out),
		Auxiliary: af,
		Zero:      out == 0,
		Sign:      out>>uint(width-1)&1 == 1,
		Overflow:  carryMSB != carryOut,
	}
}

func natLogic(group byte, a, b uint16, width int) (uint16, i8086.Flags) {
	mask := widthMask(width)
	av := uint32(a) & mask
	bv := uint32(b) & mask
	var out uint32
	switch group & 0x07 {
	case i8086.GroupAND:
		out = av & bv
	case i8086.GroupOR:
		out = av | bv
	default: // GroupXOR
		out = av ^ bv
	}
	return uint16(out), i8086.Flags{
		Parity: parityEvenLow8(out),
		Zero:   out == 0,
		Sign:   out>>uint(width-1)&1 == 1,
	}
}

func (nativeBackend) ALU(group byte, a, b uint16, width int, carryIn bool) (uint16, i8086.Flags) {
	switch group & 0x07 {
	case i8086.GroupADD:
		return natArith(a, b, width, false, false)
	case i8086.GroupADC:
		return natArith(a, b, width, carryIn, false)
	case i8086.GroupSUB, i8086.GroupCMP:
		return natArith(a, b, width, true, true)
	case i8086.GroupSBB:
		return natArith(a, b, width, !carryIn, true)
	default: // GroupAND, GroupOR, GroupXOR
		return natLogic(group, a, b, width)
	}
}

func (nativeBackend) Increment(value uint16, width int) (uint16, i8086.Flags) {
	out, f := natArith(value, 1, width, false, false)
	f.Carry = false
	return out, f
}

func (nativeBackend) Decrement(value uint16, width int) (uint16, i8086.Flags) {
	out, f := natArith(value, 1, width, true, true)
	f.Carry = false
	return out, f
}

func (nativeBackend) Mul(a, b uint16, width int, signed bool) (uint32, bool) {
	mask := widthMask(width)
	w2 := width * 2
	maskw2 := uint32(1)<<uint(w2) - 1
	av := uint32(a) & mask
	bv := uint32(b) & mask
	var prod uint32
	if signed {
		prod = uint32(signExtend(av, width)*signExtend(bv, width)) & maskw2
	} else {
		prod = (av * bv) & maskw2
	}
	hi := prod >> uint(width)
	lo := prod & mask
	var overflow bool
	switch {
	case !signed:
		overflow = hi != 0
	case lo>>uint(width-1)&1 == 1:
		overflow = hi != mask
	default:
		overflow = hi != 0
	}
	return prod, overflow
}

func (nativeBackend) Div(dividend uint32, divisor uint16, width int, signed bool) (uint16, uint16, bool) {
	mask := widthMask(width)
	w2 := width * 2
	maskw2 := uint32(1)<<uint(w2) - 1
	dv := uint32(divisor) & mask
	dd := dividend & maskw2
	if dv == 0 {
		return 0, 0, false
	}
	if !signed {
		q := dd / dv
		if q > mask {
			return 0, 0, false
		}
		return uint16(q), uint16(dd % dv), true
	}
	sdd := signExtend(dd, w2)
	sdv := signExtend(dv, width)
	q := sdd / sdv // troncamento verso zero, come IDIV
	r := sdd % sdv // resto col segno del dividendo
	lo := -(int64(1) << uint(width-1))
	hi := (int64(1) << uint(width-1)) - 1
	if q < lo || q > hi {
		return 0, 0, false
	}
	return uint16(uint32(q) & mask), uint16(uint32(r) & mask), true
}

func b2u(v bool) uint32 {
	if v {
		return 1
	}
	return 0
}

func (nativeBackend) Shift(op byte, value uint16, count byte, width int, carryIn bool) (uint16, i8086.ShiftFlags, bool) {
	mask := widthMask(width)
	v := uint32(value) & mask
	cf := carryIn
	o := op & 0x07
	for i := byte(0); i < count; i++ {
		switch o {
		case i8086.ShiftROL:
			top := v >> uint(width-1) & 1
			v = (v<<1 | top) & mask
			cf = top == 1
		case i8086.ShiftROR:
			bot := v & 1
			v = v>>1 | bot<<uint(width-1)
			cf = bot == 1
		case i8086.ShiftRCL:
			top := v >> uint(width-1) & 1
			v = (v<<1 | b2u(cf)) & mask
			cf = top == 1
		case i8086.ShiftRCR:
			bot := v & 1
			v = v>>1 | b2u(cf)<<uint(width-1)
			cf = bot == 1
		case i8086.ShiftSHR:
			cf = v&1 == 1
			v >>= 1
		case i8086.ShiftSAR:
			cf = v&1 == 1
			v = v>>1 | (v>>uint(width-1)&1)<<uint(width-1)
		default: // SHL e alias 6
			cf = v>>uint(width-1)&1 == 1
			v = v << 1 & mask
		}
	}
	res := uint16(v)
	f := i8086.ShiftFlags{
		Carry:  cf,
		Sign:   uint32(res)>>uint(width-1)&1 == 1,
		Zero:   uint32(res)&mask == 0,
		Parity: parityEvenLow8(uint32(res)),
	}
	switch o {
	case i8086.ShiftSHL, 6, i8086.ShiftROL, i8086.ShiftRCL:
		f.Overflow = f.Sign != cf
	case i8086.ShiftSHR:
		f.Overflow = value>>uint(width-1)&1 == 1
	case i8086.ShiftSAR:
		f.Overflow = false
	default: // ROR, RCR
		f.Overflow = f.Sign != (uint32(res)>>uint(width-2)&1 == 1)
	}
	return res, f, o <= i8086.ShiftRCR
}
