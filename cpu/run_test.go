package cpu

import "testing"

// loadProgram crea una CPU col backend dato, carica code a 0000:0100 e prepara
// uno stack a 0000:FFFE.
func loadProgram(backend ALUBackend, code []byte) *CPU8086 {
	c := NewCPU8086WithALU(backend)
	c.Seg[CS] = 0x0000
	c.Seg[DS] = 0x0000
	c.Seg[SS] = 0x0000
	c.Seg[ES] = 0x0000
	c.IP = 0x0100
	c.Regs[SP] = 0xFFFE
	c.Mem.(*RAM).LoadAt(PhysAddr(c.Seg[CS], c.IP), code)
	return c
}

func run(t *testing.T, backend ALUBackend, code []byte) *CPU8086 {
	t.Helper()
	c := loadProgram(backend, code)
	if _, err := c.Run(1000); err != nil {
		t.Fatalf("esecuzione fallita: %v", err)
	}
	if !c.Halted {
		t.Fatalf("la CPU non e' arrivata in HALT")
	}
	return c
}

// Somma 1..5 con un loop che usa ADD sull'ALU a gate. Atteso AX=15.
//
//	MOV AX,0 ; MOV CX,5 ; loop: ADD AX,CX ; LOOP loop ; HLT
func TestRunSum1To5(t *testing.T) {
	code := []byte{
		0xB8, 0x00, 0x00, // MOV AX,0
		0xB9, 0x05, 0x00, // MOV CX,5
		0x01, 0xC8, // ADD AX,CX
		0xE2, 0xFC, // LOOP -4
		0xF4, // HLT
	}
	for _, b := range []struct {
		name string
		be   ALUBackend
	}{{"gate", Gate}, {"native", Native}} {
		c := run(t, b.be, code)
		if c.Regs[AX] != 15 {
			t.Errorf("[%s] AX=%d, atteso 15", b.name, c.Regs[AX])
		}
		if c.Regs[CX] != 0 {
			t.Errorf("[%s] CX=%d, atteso 0", b.name, c.Regs[CX])
		}
	}
}

// Fattoriale 5! = 120 con MUL ripetute (moltiplicatore a gate). Atteso AX=120,
// DX=0.
func TestRunFactorialViaGateMul(t *testing.T) {
	code := []byte{
		0xB8, 0x01, 0x00, // MOV AX,1
		0xBB, 0x02, 0x00, 0xF7, 0xE3, // MOV BX,2 ; MUL BX
		0xBB, 0x03, 0x00, 0xF7, 0xE3, // MOV BX,3 ; MUL BX
		0xBB, 0x04, 0x00, 0xF7, 0xE3, // MOV BX,4 ; MUL BX
		0xBB, 0x05, 0x00, 0xF7, 0xE3, // MOV BX,5 ; MUL BX
		0xF4, // HLT
	}
	c := run(t, Gate, code)
	if c.Regs[AX] != 120 || c.Regs[DX] != 0 {
		t.Fatalf("AX=%d DX=%d, atteso 120/0", c.Regs[AX], c.Regs[DX])
	}
}

// PUSH/POP attraverso lo stack su SS:SP.
//
//	MOV AX,0x1234 ; PUSH AX ; POP BX ; HLT
func TestPushPop(t *testing.T) {
	code := []byte{
		0xB8, 0x34, 0x12, // MOV AX,0x1234
		0x50, // PUSH AX
		0x5B, // POP BX
		0xF4, // HLT
	}
	c := run(t, Gate, code)
	if c.Regs[BX] != 0x1234 {
		t.Errorf("BX=%#04x, atteso 0x1234", c.Regs[BX])
	}
	if c.Regs[SP] != 0xFFFE {
		t.Errorf("SP=%#04x, atteso 0xFFFE (stack bilanciato)", c.Regs[SP])
	}
}

// CMP + Jcc: conta in CL finche' BL raggiunge 3.
//
//	XOR CL,CL ; loop: INC CL ; CMP CL,3 ; JL loop ; HLT  -> CL=3
func TestCmpAndConditionalJump(t *testing.T) {
	code := []byte{
		0x30, 0xC9, // XOR CL,CL
		0xFE, 0xC1, // INC CL          (FE /0, ModRM 11 000 001)
		0x80, 0xF9, 0x03, // CMP CL,3  (80 /7, ModRM 11 111 001 = 0xF9)
		0x7C, 0xF9, // JL -7
		0xF4, // HLT
	}
	c := run(t, Gate, code)
	if c.Get8(CL) != 3 {
		t.Errorf("CL=%d, atteso 3", c.Get8(CL))
	}
}

// SHL AL,CL sull'ALU a gate: 1 << 4 = 0x10.
//
//	MOV AL,1 ; MOV CL,4 ; SHL AL,CL ; HLT
func TestShiftLeftByCL(t *testing.T) {
	code := []byte{
		0xB0, 0x01, // MOV AL,1
		0xB1, 0x04, // MOV CL,4
		0xD2, 0xE0, // SHL AL,CL   (D2 /4, ModRM 11 100 000)
		0xF4, // HLT
	}
	c := run(t, Gate, code)
	if c.Get8(AL) != 0x10 {
		t.Errorf("AL=%#02x, atteso 0x10", c.Get8(AL))
	}
}

// REP MOVSB copia un blocco da DS:SI a ES:DI.
//
//	MOV SI,0x300 ; MOV DI,0x400 ; MOV CX,5 ; REP MOVSB ; HLT
func TestRepMovsb(t *testing.T) {
	code := []byte{
		0xBE, 0x00, 0x03, // MOV SI,0x0300
		0xBF, 0x00, 0x04, // MOV DI,0x0400
		0xB9, 0x05, 0x00, // MOV CX,5
		0xF3, 0xA4, // REP MOVSB
		0xF4, // HLT
	}
	c := loadProgram(Gate, code)
	src := []byte{0x11, 0x22, 0x33, 0x44, 0x55}
	c.Mem.(*RAM).LoadAt(0x0300, src)
	if _, err := c.Run(1000); err != nil {
		t.Fatalf("esecuzione fallita: %v", err)
	}
	for i, b := range src {
		if got := c.Mem.Read8(uint32(0x0400 + i)); got != b {
			t.Errorf("dest[%d]=%#02x, atteso %#02x", i, got, b)
		}
	}
	if c.Regs[CX] != 0 {
		t.Errorf("CX=%d, atteso 0", c.Regs[CX])
	}
}

// La divisione per zero deve sollevare INT 0 e saltare al gestore puntato dal
// vettore 0. Il gestore qui e' un HLT a 0000:0200.
func TestDivByZeroRaisesInterrupt0(t *testing.T) {
	code := []byte{
		0xB8, 0x05, 0x00, // MOV AX,5
		0xB3, 0x00, // MOV BL,0
		0xF6, 0xF3, // DIV BL   (F6 /6, ModRM 11 110 011)
		0xF4, // HLT (non dovrebbe essere raggiunto)
	}
	c := loadProgram(Gate, code)
	// Vettore 0: offset 0x0200, segmento 0x0000.
	c.Mem.(*RAM).LoadAt(0x0000, []byte{0x00, 0x02, 0x00, 0x00})
	// Gestore: HLT a 0000:0200.
	c.Mem.(*RAM).LoadAt(0x0200, []byte{0xF4})
	if _, err := c.Run(1000); err != nil {
		t.Fatalf("esecuzione fallita: %v", err)
	}
	if !c.Halted || c.Seg[CS] != 0x0000 || c.IP != 0x0201 {
		t.Fatalf("non e' stato raggiunto il gestore: CS=%#04x IP=%#04x halted=%v", c.Seg[CS], c.IP, c.Halted)
	}
}

// Parita' a livello di programma: Gate e Native devono lasciare lo stesso stato
// di registri e flag dopo lo stesso programma (qui un mix di ALU, MUL e DIV con
// segno).
func TestGateVsNativeProgramState(t *testing.T) {
	code := []byte{
		0xB8, 0xF6, 0xFF, // MOV AX,0xFFF6  (-10)
		0xBB, 0x03, 0x00, // MOV BX,3
		0x99,       // CWD (estende il segno di AX in DX)
		0xF7, 0xFB, // IDIV BX   (F7 /7, ModRM 11 111 011) -> AX=-3, DX=-1
		0x05, 0x64, 0x00, // ADD AX,100
		0xF7, 0xEB, // IMUL BX   (F7 /5, ModRM 11 101 011)
		0xF4, // HLT
	}
	g := run(t, Gate, code)
	n := run(t, Native, code)
	if g.Regs != n.Regs || g.PackFlags() != n.PackFlags() {
		t.Fatalf("divergenza Gate/Native:\n gate regs=%v flags=%#04x\n nat  regs=%v flags=%#04x",
			g.Regs, g.PackFlags(), n.Regs, n.PackFlags())
	}
}
