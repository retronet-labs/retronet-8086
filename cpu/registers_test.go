package cpu

import "testing"

func TestByteHalvesShareWordStorage(t *testing.T) {
	var c CPU8086
	c.Set16(AX, 0x1234)
	if got := c.Get8(AL); got != 0x34 {
		t.Errorf("AL = %#02x, atteso 0x34", got)
	}
	if got := c.Get8(AH); got != 0x12 {
		t.Errorf("AH = %#02x, atteso 0x12", got)
	}
	c.Set8(AH, 0xAB)
	if got := c.Get16(AX); got != 0xAB34 {
		t.Errorf("AX = %#04x, atteso 0xAB34", got)
	}
	c.Set8(AL, 0xCD)
	if got := c.Get16(AX); got != 0xABCD {
		t.Errorf("AX = %#04x, atteso 0xABCD", got)
	}
}

func TestByteRegisterMapping(t *testing.T) {
	var c CPU8086
	// Ogni mezzo-registro deve toccare il word giusto.
	c.Set8(BL, 0x11)
	c.Set8(CH, 0x22)
	c.Set8(DL, 0x33)
	if c.Get16(BX) != 0x0011 {
		t.Errorf("BX = %#04x", c.Get16(BX))
	}
	if c.Get16(CX) != 0x2200 {
		t.Errorf("CX = %#04x", c.Get16(CX))
	}
	if c.Get16(DX) != 0x0033 {
		t.Errorf("DX = %#04x", c.Get16(DX))
	}
}

func TestPhysAddrSegmentation(t *testing.T) {
	cases := []struct {
		seg, off uint16
		want     uint32
	}{
		{0x0000, 0x0000, 0x00000},
		{0x1000, 0x0000, 0x10000},
		{0x1234, 0x5678, 0x179B8},                          // 0x12340 + 0x5678
		{0xFFFF, 0x000F, 0xFFFFF},                          // 0xFFFF0 + 0x000F
		{0xFFFF, 0xFFFF, (0xFFFF0 + 0xFFFF) & AddressMask}, // wrap a 20 bit
	}
	for _, c := range cases {
		if got := PhysAddr(c.seg, c.off); got != c.want {
			t.Errorf("PhysAddr(%#04x,%#04x) = %#05x, atteso %#05x", c.seg, c.off, got, c.want)
		}
	}
}

func TestFlagsPackRoundTrip(t *testing.T) {
	var c CPU8086
	c.CF = true
	c.ZF = true
	c.OF = true
	c.IF = true
	packed := c.PackFlags()
	// I bit riservati 1 e 12-15 devono risultare sempre alti.
	if packed&flagsReserved != flagsReserved {
		t.Errorf("bit riservati non impostati: %#04x", packed)
	}
	if packed&FlagCF == 0 || packed&FlagZF == 0 || packed&FlagOF == 0 || packed&FlagIF == 0 {
		t.Errorf("flag attesi mancanti: %#04x", packed)
	}
	if packed&FlagSF != 0 || packed&FlagAF != 0 {
		t.Errorf("flag spuri impostati: %#04x", packed)
	}

	var d CPU8086
	d.SetFlags(packed)
	if !d.CF || !d.ZF || !d.OF || !d.IF || d.SF || d.AF || d.PF {
		t.Errorf("round-trip flag errato: %+v", d)
	}
}
