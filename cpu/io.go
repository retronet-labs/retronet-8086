package cpu

// Ports astrae lo spazio di I/O separato dell'8086 (64 K porte). Il core ne usa
// un'implementazione no-op di default; retronet-pc vi mappera' 8259/8253/8255 ecc.
type Ports interface {
	In8(port uint16) byte
	Out8(port uint16, value byte)
}

// nullPorts e' lo spazio I/O di default: ogni lettura restituisce 0xFF (linee del
// bus a riposo), ogni scrittura e' ignorata.
type nullPorts struct{}

func (nullPorts) In8(uint16) byte   { return 0xFF }
func (nullPorts) Out8(uint16, byte) {}

// In16/Out16 compongono due accessi a 8 bit (little-endian), coerenti con il bus
// dati a 8 bit dell'8088.
func (c *CPU8086) in16(port uint16) uint16 {
	lo := uint16(c.IO.In8(port))
	hi := uint16(c.IO.In8(port + 1))
	return lo | hi<<8
}

func (c *CPU8086) out16(port uint16, value uint16) {
	c.IO.Out8(port, byte(value))
	c.IO.Out8(port+1, byte(value>>8))
}
