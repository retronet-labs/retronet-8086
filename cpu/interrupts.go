package cpu

// Interrupt consegna alla CPU un interrupt con vettore n, eseguendone la
// sequenza completa (impila FLAGS/CS/IP, azzera IF e TF, salta al gestore).
// E' il punto d'ingresso pubblico per gli interrupt hardware: la macchina lo
// chiama quando il controllore (8259 PIC) ha riconosciuto un IRQ. Sveglia anche
// la CPU da HLT (un interrupt riprende l'esecuzione).
//
// Spetta al chiamante verificare che gli interrupt siano abilitati (flag IF) per
// gli IRQ mascherabili; per gli interrupt non mascherabili (NMI) o software il
// controllo non si applica.
func (c *CPU8086) Interrupt(n byte) {
	c.Halted = false
	c.raiseInterrupt(n)
}

// InterruptsEnabled indica se gli interrupt mascherabili sono abilitati (IF=1).
func (c *CPU8086) InterruptsEnabled() bool { return c.IF }

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
