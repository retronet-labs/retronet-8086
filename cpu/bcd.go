package cpu

import "github.com/retronet-labs/retronet-hardware/bridge/i8086"

// Le istruzioni di aggiustamento decimale e ASCII dell'8086. Dove serve
// aritmetica vera (AAM/AAD) si appoggiano alla ALU a gate (Div/Mul); SF/ZF/PF si
// ricavano con una OR a gate (setSZP).

// setSZP imposta SF/ZF/PF dal valore usando la OR a gate (che lascia il valore
// invariato e azzera CF/OF).
func (c *CPU8086) setSZP(v uint16, w bool) {
	_, f := c.backend().ALU(i8086.GroupOR, v, v, width(w), false)
	c.SF, c.ZF, c.PF = f.Sign, f.Zero, f.Parity
}

// daa (0x27): aggiustamento decimale dopo addizione.
func (c *CPU8086) daa() {
	oldAL := c.Get8(AL)
	oldCF := c.CF
	al := oldAL
	cf := false
	if al&0x0F > 9 || c.AF {
		al += 6
		cf = oldCF || al < oldAL // riporto/wrap dall'aggiunta di 6
		c.AF = true
	} else {
		c.AF = false
	}
	if oldAL > 0x99 || oldCF {
		al += 0x60
		cf = true
	}
	c.Set8(AL, al)
	c.CF = cf
	c.setSZP(uint16(al), false)
}

// das (0x2F): aggiustamento decimale dopo sottrazione.
func (c *CPU8086) das() {
	oldAL := c.Get8(AL)
	oldCF := c.CF
	al := oldAL
	cf := false
	if al&0x0F > 9 || c.AF {
		cf = oldCF || al < 6 // prestito dalla sottrazione di 6
		al -= 6
		c.AF = true
	} else {
		c.AF = false
	}
	if oldAL > 0x99 || oldCF {
		al -= 0x60
		cf = true
	}
	c.Set8(AL, al)
	c.CF = cf
	c.setSZP(uint16(al), false)
}

// aaa (0x37): aggiustamento ASCII dopo addizione.
func (c *CPU8086) aaa() {
	if c.Get8(AL)&0x0F > 9 || c.AF {
		c.Set8(AL, c.Get8(AL)+6)
		c.Set8(AH, c.Get8(AH)+1)
		c.AF, c.CF = true, true
	} else {
		c.AF, c.CF = false, false
	}
	c.Set8(AL, c.Get8(AL)&0x0F)
}

// aas (0x3F): aggiustamento ASCII dopo sottrazione.
func (c *CPU8086) aas() {
	if c.Get8(AL)&0x0F > 9 || c.AF {
		c.Set8(AL, c.Get8(AL)-6)
		c.Set8(AH, c.Get8(AH)-1)
		c.AF, c.CF = true, true
	} else {
		c.AF, c.CF = false, false
	}
	c.Set8(AL, c.Get8(AL)&0x0F)
}

// aam (0xD4 ib): AH = AL / base, AL = AL % base (divisione a gate). #DE se base==0.
func (c *CPU8086) aam() {
	base := c.fetch8()
	q, r, ok := c.backend().Div(uint32(c.Get8(AL)), uint16(base), 8, false)
	if !ok {
		c.raiseInterrupt(0)
		return
	}
	c.Set8(AH, byte(q))
	c.Set8(AL, byte(r))
	c.setSZP(uint16(c.Get8(AL)), false)
}

// aad (0xD5 ib): AL = (AH * base + AL) & 0xFF, AH = 0 (moltiplicazione a gate).
func (c *CPU8086) aad() {
	base := c.fetch8()
	prod, _ := c.backend().Mul(uint16(c.Get8(AH)), uint16(base), 8, false)
	sum, _ := c.backend().ALU(i8086.GroupADD, uint16(prod), uint16(c.Get8(AL)), 8, false)
	c.Set8(AL, byte(sum))
	c.Set8(AH, 0)
	c.setSZP(uint16(c.Get8(AL)), false)
}
