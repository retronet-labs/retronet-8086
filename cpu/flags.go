package cpu

// PackFlags impacchetta i flag booleani nel registro FLAGS a 16 bit, applicando
// i bit riservati che l'8086 espone sempre a 1 (bit 1 e 12-15). E' l'immagine
// che vedono PUSHF/IRET e i vettori di conformita'.
func (c *CPU8086) PackFlags() uint16 {
	var f uint16 = flagsReserved
	for _, fl := range []struct {
		on   bool
		mask uint16
	}{
		{c.CF, FlagCF}, {c.PF, FlagPF}, {c.AF, FlagAF}, {c.ZF, FlagZF},
		{c.SF, FlagSF}, {c.TF, FlagTF}, {c.IF, FlagIF}, {c.DF, FlagDF}, {c.OF, FlagOF},
	} {
		if fl.on {
			f |= fl.mask
		}
	}
	return f
}

// SetFlags scompone un valore FLAGS a 16 bit nei campi booleani, ignorando i bit
// riservati.
func (c *CPU8086) SetFlags(v uint16) {
	c.CF = v&FlagCF != 0
	c.PF = v&FlagPF != 0
	c.AF = v&FlagAF != 0
	c.ZF = v&FlagZF != 0
	c.SF = v&FlagSF != 0
	c.TF = v&FlagTF != 0
	c.IF = v&FlagIF != 0
	c.DF = v&FlagDF != 0
	c.OF = v&FlagOF != 0
}
