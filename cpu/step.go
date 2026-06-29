package cpu

// Step esegue una singola istruzione: raccoglie gli eventuali prefissi, poi
// decodifica ed esegue l'opcode. Restituisce un *UnimplementedError sugli opcode
// non ancora gestiti. Se la CPU e' in HALT non fa nulla.
//
// Trap Flag (single-step): se TF e' attivo all'INIZIO dell'istruzione, al termine
// la CPU genera l'interrupt di single-step (INT 1). Il campionamento "prima"
// dell'esecuzione e' il comportamento del silicio (un latch interno): un'istruzione
// che ABILITA TF (POPF/IRET) non produce il trap su se stessa ma a partire dalla
// successiva, e la sequenza d'interrupt — che azzera TF — non ricorre all'infinito.
// E' cio' che permette ai debugger DOS (il comando T di DEBUG.COM) di tracciare.
func (c *CPU8086) Step() error {
	if c.Halted {
		return nil
	}
	trap := c.TF
	if err := c.decodeExecute(); err != nil {
		return err
	}
	if trap && !c.Halted {
		c.raiseInterrupt(1)
	}
	return nil
}

// decodeExecute raccoglie i prefissi e decodifica/esegue l'opcode.
func (c *CPU8086) decodeExecute() error {
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
		case 0xF0, 0xF1: // 0xF1 undocumented: alias of LOCK on NMOS 8086/8088
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
