package cpu

import "testing"

func TestDAA(t *testing.T) {
	cases := []struct {
		al, want byte
		af, cf   bool
		wantCF   bool
	}{
		{0x3A, 0x40, false, false, false}, // nibble basso > 9 -> +6
		{0x9A, 0x00, false, false, true},  // entrambi gli aggiustamenti -> 0x00, CF
		{0x1F, 0x25, false, false, false}, // 0x1F -> +6 = 0x25
	}
	for _, k := range cases {
		c := NewCPU8086()
		c.Set8(AL, k.al)
		c.AF, c.CF = k.af, k.cf
		c.daa()
		if c.Get8(AL) != k.want || c.CF != k.wantCF {
			t.Errorf("DAA(%#02x) = %#02x CF=%v, atteso %#02x CF=%v", k.al, c.Get8(AL), c.CF, k.want, k.wantCF)
		}
	}
}

func TestAAMAAD(t *testing.T) {
	c := NewCPU8086()
	c.Set8(AL, 77) // 0x4D
	c.aamWith(t, 10)
	if c.Get8(AH) != 7 || c.Get8(AL) != 7 {
		t.Fatalf("AAM: AH=%d AL=%d, atteso 7/7", c.Get8(AH), c.Get8(AL))
	}
	// AAD inverte: da AH=7,AL=7 base 10 -> AL=77, AH=0
	c.aadWith(t, 10)
	if c.Get8(AL) != 77 || c.Get8(AH) != 0 {
		t.Fatalf("AAD: AL=%d AH=%d, atteso 77/0", c.Get8(AL), c.Get8(AH))
	}
}

// aamWith/aadWith iniettano l'immediato come se fosse nel flusso d'istruzioni.
func (c *CPU8086) aamWith(t *testing.T, base byte) {
	t.Helper()
	c.Seg[CS], c.IP = 0, 0x100
	c.Mem.Write8(PhysAddr(0, 0x100), base)
	c.aam()
}

func (c *CPU8086) aadWith(t *testing.T, base byte) {
	t.Helper()
	c.Seg[CS], c.IP = 0, 0x100
	c.Mem.Write8(PhysAddr(0, 0x100), base)
	c.aad()
}

func TestAAA(t *testing.T) {
	c := NewCPU8086()
	c.Set16(AX, 0x000F) // AL=0x0F
	c.aaa()
	// 0x0F basso>9 -> AL=0x15&0x0F=5, AH+1=1, CF/AF=1
	if c.Get8(AL) != 0x05 || c.Get8(AH) != 0x01 || !c.CF || !c.AF {
		t.Fatalf("AAA: AX=%#04x CF=%v AF=%v", c.Get16(AX), c.CF, c.AF)
	}
}
