package cpu

import (
	"testing"

	"github.com/retronet-labs/retronet-hardware/bridge/i8086"
)

var aluGroups = []byte{
	i8086.GroupADD, i8086.GroupOR, i8086.GroupADC, i8086.GroupSBB,
	i8086.GroupAND, i8086.GroupSUB, i8086.GroupXOR, i8086.GroupCMP,
}

// TestGateVsNativeALUDifferential e' la garanzia centrale: il backend a porte e
// quello nativo devono restituire risultato e flag identici su ogni ingresso. A
// 8 bit la copertura e' esaustiva; a 16 bit e' campionata.
func TestGateVsNativeALUDifferential(t *testing.T) {
	for _, g := range aluGroups {
		for a := 0; a <= 0xFF; a++ {
			for b := 0; b <= 0xFF; b++ {
				for _, cin := range []bool{false, true} {
					gr, gf := Gate.ALU(g, uint16(a), uint16(b), 8, cin)
					nr, nf := Native.ALU(g, uint16(a), uint16(b), 8, cin)
					if gr != nr || gf != nf {
						t.Fatalf("ALU8 g=%d a=%#x b=%#x cin=%v: gate(%#x,%+v) native(%#x,%+v)",
							g, a, b, cin, gr, gf, nr, nf)
					}
				}
			}
		}
		const step = 277
		for a := 0; a <= 0xFFFF; a += step {
			for b := 0; b <= 0xFFFF; b += step {
				for _, cin := range []bool{false, true} {
					gr, gf := Gate.ALU(g, uint16(a), uint16(b), 16, cin)
					nr, nf := Native.ALU(g, uint16(a), uint16(b), 16, cin)
					if gr != nr || gf != nf {
						t.Fatalf("ALU16 g=%d a=%#x b=%#x cin=%v: gate(%#x,%+v) native(%#x,%+v)",
							g, a, b, cin, gr, gf, nr, nf)
					}
				}
			}
		}
	}
}

func TestGateVsNativeIncDec(t *testing.T) {
	for _, width := range []int{8, 16} {
		mask := 1<<uint(width) - 1
		for v := 0; v <= mask; v += 1 + v/4096 {
			gi, gfi := Gate.Increment(uint16(v), width)
			ni, nfi := Native.Increment(uint16(v), width)
			if gi != ni || gfi != nfi {
				t.Fatalf("INC width=%d v=%#x: gate(%#x,%+v) native(%#x,%+v)", width, v, gi, gfi, ni, nfi)
			}
			gd, gfd := Gate.Decrement(uint16(v), width)
			nd, nfd := Native.Decrement(uint16(v), width)
			if gd != nd || gfd != nfd {
				t.Fatalf("DEC width=%d v=%#x: gate(%#x,%+v) native(%#x,%+v)", width, v, gd, gfd, nd, nfd)
			}
		}
	}
}

func TestGateVsNativeMul(t *testing.T) {
	for _, signed := range []bool{false, true} {
		for a := 0; a <= 0xFF; a++ {
			for b := 0; b <= 0xFF; b++ {
				gp, go_ := Gate.Mul(uint16(a), uint16(b), 8, signed)
				np, no := Native.Mul(uint16(a), uint16(b), 8, signed)
				if gp != np || go_ != no {
					t.Fatalf("Mul8 signed=%v a=%#x b=%#x: gate(%#x,%v) native(%#x,%v)", signed, a, b, gp, go_, np, no)
				}
			}
		}
		const step = 433
		for a := 0; a <= 0xFFFF; a += step {
			for b := 0; b <= 0xFFFF; b += step {
				gp, go_ := Gate.Mul(uint16(a), uint16(b), 16, signed)
				np, no := Native.Mul(uint16(a), uint16(b), 16, signed)
				if gp != np || go_ != no {
					t.Fatalf("Mul16 signed=%v a=%#x b=%#x: gate(%#x,%v) native(%#x,%v)", signed, a, b, gp, go_, np, no)
				}
			}
		}
	}
}

func TestGateVsNativeDiv(t *testing.T) {
	for _, signed := range []bool{false, true} {
		const step8 = 131
		for dd := 0; dd <= 0xFFFF; dd += step8 {
			for dv := 0; dv <= 0xFF; dv++ {
				gq, gr, gok := Gate.Div(uint32(dd), uint16(dv), 8, signed)
				nq, nr, nok := Native.Div(uint32(dd), uint16(dv), 8, signed)
				if gok != nok || (gok && (gq != nq || gr != nr)) {
					t.Fatalf("Div8 signed=%v dd=%#x dv=%#x: gate(%#x,%#x,%v) native(%#x,%#x,%v)",
						signed, dd, dv, gq, gr, gok, nq, nr, nok)
				}
			}
		}
		const stepDD = 2796203
		const stepDV = 521
		for dd := uint64(0); dd <= 0xFFFFFFFF; dd += stepDD {
			for dv := 0; dv <= 0xFFFF; dv += stepDV {
				gq, gr, gok := Gate.Div(uint32(dd), uint16(dv), 16, signed)
				nq, nr, nok := Native.Div(uint32(dd), uint16(dv), 16, signed)
				if gok != nok || (gok && (gq != nq || gr != nr)) {
					t.Fatalf("Div16 signed=%v dd=%#x dv=%#x: gate(%#x,%#x,%v) native(%#x,%#x,%v)",
						signed, dd, dv, gq, gr, gok, nq, nr, nok)
				}
			}
		}
	}
}
