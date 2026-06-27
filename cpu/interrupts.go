package cpu

// raiseInterrupt esegue la sequenza di interrupt software/hardware dell'8086:
// impila FLAGS, azzera IF e TF, impila CS e IP, quindi carica il nuovo CS:IP dal
// vettore situato a n*4 nella tabella all'inizio della memoria (0x0000).
func (c *CPU8086) raiseInterrupt(n byte) {
	c.push16(c.PackFlags())
	c.IF = false
	c.TF = false
	c.push16(c.Seg[CS])
	c.push16(c.IP)
	vector := uint32(n) * 4
	c.IP = c.memRead16Phys(vector)
	c.Seg[CS] = c.memRead16Phys(vector + 2)
}

// iret ripristina IP, CS e FLAGS dallo stack (ritorno da interrupt).
func (c *CPU8086) iret() {
	c.IP = c.pop16()
	c.Seg[CS] = c.pop16()
	c.SetFlags(c.pop16())
}

// memRead16Phys legge una parola little-endian a un indirizzo fisico assoluto
// (usata per i vettori d'interrupt).
func (c *CPU8086) memRead16Phys(addr uint32) uint16 {
	lo := uint16(c.Mem.Read8(addr & AddressMask))
	hi := uint16(c.Mem.Read8((addr + 1) & AddressMask))
	return lo | hi<<8
}
