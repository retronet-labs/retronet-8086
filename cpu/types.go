// Package cpu implementa il core dell'Intel 8086/8088 in real mode: registri,
// segmentazione, decodifica e le due ALU intercambiabili (a porte logiche e
// native), nello stesso stile dei moduli 4004/8008/8080 di RetroNet.
package cpu

// Reg16 identifica i registri generali a 16 bit nell'ordine del campo reg/rm del
// ModR/M quando w=1.
type Reg16 int

const (
	AX Reg16 = iota
	CX
	DX
	BX
	SP
	BP
	SI
	DI
)

// Reg8 identifica i mezzi-registri a 8 bit nell'ordine del campo reg/rm quando
// w=0: AL CL DL BL AH CH DH BH.
type Reg8 int

const (
	AL Reg8 = iota
	CL
	DL
	BL
	AH
	CH
	DH
	BH
)

// Sreg identifica i registri di segmento nell'ordine del campo sreg.
type Sreg int

const (
	ES Sreg = iota
	CS
	SS
	DS
)

// Posizione dei flag nel registro FLAGS dell'8086.
const (
	FlagCF = 1 << 0  // Carry
	FlagPF = 1 << 2  // Parity (sugli 8 bit bassi)
	FlagAF = 1 << 4  // Auxiliary Carry (mezzo-byte)
	FlagZF = 1 << 6  // Zero
	FlagSF = 1 << 7  // Sign
	FlagTF = 1 << 8  // Trap (single-step)
	FlagIF = 1 << 9  // Interrupt enable
	FlagDF = 1 << 10 // Direction (istruzioni stringa)
	FlagOF = 1 << 11 // Overflow
)

// flagsReserved sono i bit che sull'8086 si leggono sempre a 1 (bit 1 e 12-15).
const flagsReserved = 0xF002

// flagsMask raccoglie i bit di flag effettivamente usati dall'8086.
const flagsMask = FlagCF | FlagPF | FlagAF | FlagZF | FlagSF | FlagTF | FlagIF | FlagDF | FlagOF

// AddressMask copre lo spazio indirizzabile a 20 bit (1 MB) dell'8086.
const AddressMask uint32 = 0xFFFFF
