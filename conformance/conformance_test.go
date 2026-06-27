package conformance

import (
	"testing"

	"github.com/retronet-labs/retronet-8086/cpu"
)

func TestConformanceAllPass(t *testing.T) {
	for _, be := range []struct {
		name string
		b    cpu.ALUBackend
	}{{"gate", cpu.Gate}, {"native", cpu.Native}} {
		res := Run(be.b)
		if res.Failed() != 0 {
			for _, c := range res.Cases {
				if !c.OK {
					t.Errorf("[%s] %s: %s", be.name, c.Name, c.Detail)
				}
			}
		}
	}
}
