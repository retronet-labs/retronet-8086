package cpu

import "testing"

func TestDisassemble(t *testing.T) {
	cases := []struct {
		bytes    []byte
		wantText string
		wantLen  int
	}{
		{[]byte{0xB8, 0x34, 0x12}, "mov ax, 0x1234", 3},
		{[]byte{0xB4, 0x09}, "mov ah, 0x09", 2},
		{[]byte{0x01, 0xC8}, "add ax, cx", 2},
		{[]byte{0x88, 0x07}, "mov [bx], al", 2},           // mod00 rm=bx reg=al
		{[]byte{0x8B, 0x46, 0xFE}, "mov ax, [bp-0x2]", 3}, // mod01 rm=bp disp8 -2
		{[]byte{0x83, 0xC3, 0x05}, "add bx, 0x05", 3},     // grp1 /0 imm8 esteso
		{[]byte{0xF6, 0x33}, "div byte [bp+di]", 2},       // F6 /6
		{[]byte{0xD2, 0xE0}, "shl al, cl", 2},
		{[]byte{0xD1, 0xF3}, "setmo bx, 1", 2}, // D1 /6 = SETMO
		{[]byte{0xFF, 0x37}, "push word [bx]", 2},
		{[]byte{0xEB, 0xFE}, "jmp 0x0100", 2}, // rel8 -2 da 0x0102
		{[]byte{0x9A, 0x00, 0x00, 0x00, 0x30}, "call 0x3000:0x0000", 5},
		{[]byte{0x26, 0x8A, 0x07}, "mov al, [es:bx]", 3}, // override ES
		{[]byte{0xF3, 0xA4}, "rep movsb", 2},
		{[]byte{0xCD, 0x21}, "int 0x21", 2},
		{[]byte{0xF4}, "hlt", 1},
	}
	for _, k := range cases {
		c := NewCPU8086()
		c.Mem.(*RAM).LoadAt(PhysAddr(0, 0x100), k.bytes)
		text, length := c.Disassemble(0, 0x100)
		if text != k.wantText || length != k.wantLen {
			t.Errorf("% X: got (%q, %d), atteso (%q, %d)", k.bytes, text, length, k.wantText, k.wantLen)
		}
	}
}
