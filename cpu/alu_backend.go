package cpu

import "github.com/retronet-labs/retronet-hardware/bridge/i8086"

// ALUBackend astrae il motore aritmetico-logico dell'8086. RetroNet ne offre due
// implementazioni intercambiabili con semantica identica, flag compresi:
//
//   - Gate: l'ALU costruita dalle sole porte logiche di retronet-logic, raggiunta
//     tramite il bridge i8086. E' il default e dimostra che la CPU calcola su un
//     datapath fatto di gate (moltiplicazione e divisione comprese, composte come
//     shift-and-add e divisione a ripristino sul sommatore a gate).
//   - Native: la stessa semantica con gli operatori di Go. Piu' veloce, serve da
//     oracolo del test differenziale TestGateVsNativeALUDifferential.
//
// I due backend devono restituire lo stesso risultato e gli stessi flag su ogni
// ingresso.
type ALUBackend interface {
	// ALU esegue uno degli otto gruppi 8086 (ADD/OR/ADC/SBB/AND/SUB/XOR/CMP) su a
	// e b alla larghezza width (8 o 16), con carry entrante carryIn.
	ALU(group byte, a, b uint16, width int, carryIn bool) (uint16, i8086.Flags)
	// Increment esegue value+1 (semantica INC: il Carry non cambia).
	Increment(value uint16, width int) (uint16, i8086.Flags)
	// Decrement esegue value-1 (semantica DEC: il Carry non cambia).
	Decrement(value uint16, width int) (uint16, i8086.Flags)
	// Mul moltiplica a*b restituendo il prodotto a 2*width bit e l'overflow
	// (meta' alta significativa) usato da MUL/IMUL per CF e OF.
	Mul(a, b uint16, width int, signed bool) (product uint32, overflow bool)
	// Div divide il dividendo (2*width bit) per divisor, con ok=false in caso di
	// errore di divisione (#DE): divisore nullo o quoziente fuori intervallo.
	Div(dividend uint32, divisor uint16, width int, signed bool) (quot, rem uint16, ok bool)
}

// Gate e' il backend costruito dalle porte logiche (default).
var Gate ALUBackend = gateBackend{}

// Native e' il backend con operatori Go: oracolo del differenziale verso Gate.
var Native ALUBackend = nativeBackend{}

// gateBackend inoltra ogni operazione all'ALU a porte tramite il bridge i8086.
type gateBackend struct{}

func (gateBackend) ALU(group byte, a, b uint16, width int, carryIn bool) (uint16, i8086.Flags) {
	return i8086.ALU(group, a, b, width, carryIn)
}

func (gateBackend) Increment(value uint16, width int) (uint16, i8086.Flags) {
	return i8086.Increment(value, width)
}

func (gateBackend) Decrement(value uint16, width int) (uint16, i8086.Flags) {
	return i8086.Decrement(value, width)
}

func (gateBackend) Mul(a, b uint16, width int, signed bool) (uint32, bool) {
	return i8086.Mul(a, b, width, signed)
}

func (gateBackend) Div(dividend uint32, divisor uint16, width int, signed bool) (uint16, uint16, bool) {
	return i8086.Div(dividend, divisor, width, signed)
}

// SetALU sceglie il backend aritmetico-logico (Gate o Native).
func (c *CPU8086) SetALU(b ALUBackend) { c.alu = b }

// backend restituisce il motore ALU corrente, con Gate come default.
func (c *CPU8086) backend() ALUBackend {
	if c.alu == nil {
		return Gate
	}
	return c.alu
}
