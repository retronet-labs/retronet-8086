package cpu

// Step esegue una singola istruzione: raccoglie gli eventuali prefissi, poi
// decodifica ed esegue l'opcode. Restituisce un *UnimplementedError sugli opcode
// non ancora gestiti. Se la CPU e' in HALT non fa nulla.
func (c *CPU8086) Step() error {
	if c.Halted {
		return nil
	}
	var pfx prefixes
	for {
		op := c.fetch8()
		switch op {
		case 0x26:
			pfx.segOverride, pfx.hasSeg = ES, true
		case 0x2E:
			pfx.segOverride, pfx.hasSeg = CS, true
		case 0x36:
			pfx.segOverride, pfx.hasSeg = SS, true
		case 0x3E:
			pfx.segOverride, pfx.hasSeg = DS, true
		case 0xF0:
			pfx.lock = true
		case 0xF2, 0xF3:
			pfx.rep = op
		default:
			return c.execute(op, pfx)
		}
	}
}

// Run esegue fino a maxSteps istruzioni o finche' la CPU non va in HALT.
// Restituisce il numero di istruzioni eseguite e l'eventuale errore.
func (c *CPU8086) Run(maxSteps int) (int, error) {
	for i := 0; i < maxSteps; i++ {
		if c.Halted {
			return i, nil
		}
		if err := c.Step(); err != nil {
			return i, err
		}
	}
	return maxSteps, nil
}
