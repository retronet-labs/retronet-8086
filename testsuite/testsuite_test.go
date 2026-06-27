package testsuite

import (
	"os"
	"testing"

	"github.com/retronet-labs/retronet-8086/cpu"
)

// vettore sintetico nel formato SingleStepTests: MOV AX,0x1234 a 0000:0100.
// 0xB8,0x34,0x12 -> AX=0x1234, IP avanza a 0x0103. flags iniziali = 0xF002.
const movAXVector = `[
  {
    "name": "mov ax,0x1234",
    "initial": {
      "regs": {"ax":0, "ip":256, "cs":0, "flags":61442},
      "ram": [[256,184],[257,52],[258,18]]
    },
    "final": {
      "regs": {"ax":4660, "ip":259},
      "ram": [[256,184],[257,52],[258,18]]
    }
  }
]`

func TestHarnessPassesValidVector(t *testing.T) {
	for _, be := range []cpu.ALUBackend{cpu.Gate, cpu.Native} {
		r, err := RunBytes([]byte(movAXVector), Options{Backend: be})
		if err != nil {
			t.Fatalf("RunBytes: %v", err)
		}
		if r.Total != 1 || r.Passed != 1 {
			t.Fatalf("atteso 1/1, ottenuto %d/%d, errori: %v", r.Passed, r.Total, r.Failures)
		}
	}
}

// L'harness deve rilevare uno stato finale sbagliato.
func TestHarnessDetectsWrongFinal(t *testing.T) {
	bad := `[
      {
        "name": "mov ax sbagliato",
        "initial": {"regs": {"ip":256,"cs":0,"flags":61442}, "ram": [[256,184],[257,52],[258,18]]},
        "final": {"regs": {"ax":9999, "ip":259}, "ram": []}
      }
    ]`
	r, err := RunBytes([]byte(bad), Options{})
	if err != nil {
		t.Fatalf("RunBytes: %v", err)
	}
	if r.Failed() != 1 {
		t.Fatalf("atteso 1 fallimento, ottenuto %d (failures: %v)", r.Failed(), r.Failures)
	}
}

// Esegue il dataset reale SingleStepTests se RETRONET_8088_TESTS punta alla
// cartella dei vettori; altrimenti salta. Confronta i flag con una maschera che
// ignora i bit lasciati indefiniti dall'8086 (AF su shift logici, ecc.).
func TestSingleStepTests8088(t *testing.T) {
	dir := os.Getenv("RETRONET_8088_TESTS")
	if dir == "" {
		t.Skip("imposta RETRONET_8088_TESTS alla cartella dei vettori SingleStepTests/8088")
	}
	res, err := RunDir(dir, Options{Backend: cpu.Gate})
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	t.Logf("SingleStepTests 8088: %d/%d passati (%d falliti)", res.Passed, res.Total, res.Failed())
	if res.Passed == 0 {
		t.Fatalf("nessun vettore passato: probabile problema di formato o di percorso")
	}
	for i, f := range res.Failures {
		if i >= 20 {
			t.Logf("... e altri %d fallimenti", res.Failed()-20)
			break
		}
		t.Log(f)
	}
}
