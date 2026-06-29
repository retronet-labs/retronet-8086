package cpu

import "testing"

// setVector imposta il vettore d'interrupt n (offset+segmento) nella IVT a 0x0000.
func setVector(c *CPU8086, n byte, seg, off uint16) {
	c.Mem.(*RAM).LoadAt(uint32(n)*4, []byte{
		byte(off), byte(off >> 8), byte(seg), byte(seg >> 8),
	})
}

// Con TF=1 la CPU, al termine dell'istruzione, deve generare l'interrupt di
// single-step (INT 1): salto al gestore, frame impilato con i flag (TF incluso) e
// l'IP di ritorno, e TF/IF azzerati dalla sequenza.
func TestTrapFlagSingleStep(t *testing.T) {
	c := loadProgram(Native, []byte{0x90}) // NOP a 0000:0100
	setVector(c, 1, 0x0000, 0x0200)        // INT1 -> 0000:0200
	c.TF = true
	c.IF = true

	if err := c.Step(); err != nil {
		t.Fatalf("Step: %v", err)
	}
	if c.Seg[CS] != 0x0000 || c.IP != 0x0200 {
		t.Fatalf("non vettorizzato al gestore INT1: CS:IP=%04X:%04X", c.Seg[CS], c.IP)
	}
	if c.TF {
		t.Error("TF doveva essere azzerato dalla sequenza d'interrupt")
	}
	if c.IF {
		t.Error("IF doveva essere azzerato dalla sequenza d'interrupt")
	}
	// Frame: [SP]=IP ritorno, [SP+2]=CS, [SP+4]=FLAGS.
	if ret := c.readMem16(c.Seg[SS], c.Regs[SP]); ret != 0x0101 {
		t.Errorf("IP di ritorno = %04X, atteso 0101 (dopo la NOP)", ret)
	}
	if f := c.readMem16(c.Seg[SS], c.Regs[SP]+4); f&FlagTF == 0 {
		t.Errorf("i flag impilati (%04X) dovevano avere TF=1", f)
	}
}

// Un'istruzione che ABILITA TF (POPF) non deve fare trap su se stessa, ma solo a
// partire dall'istruzione successiva: il TF e' campionato PRIMA dell'esecuzione.
func TestTrapFlagDeferredAfterEnable(t *testing.T) {
	c := loadProgram(Native, []byte{0x9D, 0x90}) // POPF ; NOP
	setVector(c, 1, 0x0000, 0x0200)
	c.writeMem16(c.Seg[SS], c.Regs[SP], flagsReserved|FlagTF) // valore che POPF carichera'
	c.TF = false

	if err := c.Step(); err != nil { // POPF
		t.Fatalf("Step POPF: %v", err)
	}
	if !c.TF {
		t.Fatal("POPF doveva impostare TF")
	}
	if c.IP != 0x0101 {
		t.Fatalf("dopo POPF IP=%04X, atteso 0101 (nessun trap immediato)", c.IP)
	}

	if err := c.Step(); err != nil { // NOP, ora con TF=1
		t.Fatalf("Step NOP: %v", err)
	}
	if c.IP != 0x0200 {
		t.Fatalf("la NOP con TF=1 doveva fare trap (IP=%04X, atteso 0200)", c.IP)
	}
}

// INT 3 (0xCC) vettorizza tramite la voce 3 della IVT e impila l'IP dopo l'opcode.
func TestINT3(t *testing.T) {
	c := loadProgram(Native, []byte{0xCC}) // INT3 a 0000:0100
	setVector(c, 3, 0x0000, 0x0300)

	if err := c.Step(); err != nil {
		t.Fatalf("Step: %v", err)
	}
	if c.Seg[CS] != 0x0000 || c.IP != 0x0300 {
		t.Fatalf("INT3 non vettorizzato: CS:IP=%04X:%04X", c.Seg[CS], c.IP)
	}
	if ret := c.readMem16(c.Seg[SS], c.Regs[SP]); ret != 0x0101 {
		t.Errorf("IP di ritorno = %04X, atteso 0101", ret)
	}
}
