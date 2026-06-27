package cpu

// Bus astrae lo spazio di memoria fisico visto dalla CPU. Il core ne usa una RAM
// piatta da 1 MB (RAM); retronet-pc potra' sostituirla con un bus mappato
// (RAM convenzionale + video RAM + ROM del BIOS) implementando questa interfaccia.
type Bus interface {
	Read8(addr uint32) byte
	Write8(addr uint32, value byte)
}

// RAM e' una memoria piatta da 1 MB con indirizzamento a 20 bit (wrap a 0xFFFFF).
type RAM struct {
	data [1 << 20]byte
}

// NewRAM crea una RAM azzerata da 1 MB.
func NewRAM() *RAM { return &RAM{} }

// Read8 legge un byte all'indirizzo fisico (mascherato a 20 bit).
func (m *RAM) Read8(addr uint32) byte { return m.data[addr&AddressMask] }

// Write8 scrive un byte all'indirizzo fisico (mascherato a 20 bit).
func (m *RAM) Write8(addr uint32, value byte) { m.data[addr&AddressMask] = value }

// LoadAt copia i byte a partire dall'indirizzo fisico dato (per caricare ROM o
// programmi raw nei test e nella CLI).
func (m *RAM) LoadAt(addr uint32, data []byte) {
	for i, b := range data {
		m.Write8(addr+uint32(i), b)
	}
}
