package cpu

import "fmt"

// UnimplementedError segnala un opcode non ancora gestito dal core, con il punto
// in cui e' stato incontrato.
type UnimplementedError struct {
	Opcode byte
	CS, IP uint16
}

func (e *UnimplementedError) Error() string {
	return fmt.Sprintf("opcode non implementato 0x%02X a %04X:%04X", e.Opcode, e.CS, e.IP)
}
