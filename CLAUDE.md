# CLAUDE.md — retronet-8086

Emulatore della CPU **Intel 8086/8088** in real mode (Go), motore del futuro
`retronet-pc` (IBM PC/XT). Stessa filosofia degli altri core RetroNet: core
importabile, **due ALU intercambiabili** (a porte logiche / native), test e
documentazione in italiano. Panoramica utente: [README.md](README.md);
architettura: [docs/architettura.md](docs/architettura.md).

## Setup su una macchina nuova (handoff)

1. Clona i repo come cartelle **sibling** sotto la stessa radice:
   ```
   work/source/
   ├── retronet-logic/      (github.com/retronet-labs/retronet-logic)
   ├── retronet-hardware/   (bridge i8086, da v0.7.1)
   └── retronet-8086/       (questo repo)
   ```
   Un clone pulito compila già dalle **versioni pubblicate** (go.sum presente);
   i sibling servono solo per co-sviluppare logic/hardware/8086 insieme.
2. Workspace di sviluppo locale (`go.work` **non versionato**, va ricreato):
   ```sh
   go work init . ../retronet-hardware ../retronet-logic
   ```
3. **SingleStepTests (TomHarte)** — dataset di validazione, **fuori dal repo**:
   scarica i file `v2/*.json.gz` da github.com/SingleStepTests/8088 in una
   cartella e passala con `-testsuite <dir>` (vedi sotto). Senza dataset, i test
   normali (`go test ./...`) restano verdi.

## Comandi

- Test: `go test ./...` (richiede `go.work`)
- Formattazione: `gofmt -w .` ; Analisi: `go vet ./...`
- CLI:
  ```sh
  go run ./cmd/retronet-8086 -profiles                # elenca 8086/8088
  go run ./cmd/retronet-8086 -conformance             # batteria sintetica
  go run ./cmd/retronet-8086 -bin prog.bin -trace     # esegue con trace disassemblato
  go run ./cmd/retronet-8086 -bin prog.bin -alu native
  go run ./cmd/retronet-8086 -testsuite <dir-vettori> # SingleStepTests 8088
  ```

## Componenti (`cpu/`)

- **Registri/segmentazione**: AX..DI con mezzi-registri AL/AH…, CS/DS/SS/ES, IP,
  FLAGS a 16 bit; indirizzo fisico `(seg<<4 + off) & 0xFFFFF` (wrap a 1 MB).
- **Backend ALU** (`alu_backend.go`): `Gate` (default, via `bridge/i8086`) e
  `Native` (operatori Go, oracolo). Garanzia: `TestGateVsNativeALUDifferential`
  (esaustivo a 8 bit, campionato a 16). MUL/DIV/shift su Gate sono composti dai
  primitivi a gate del bridge.
- **Profili** `Profile8086`/`Profile8088` (default 8088, IBM XT): cambiano bus e
  coda di prefetch (timing), non la semantica.
- **Decoder/execute**: ModR/M a 16 bit + override di segmento; tabella opcode +
  gruppi (`80`-`83`, `F6`/`F7`, `FE`/`FF`, `D0`-`D3`). `disasm.go` è simmetrico
  al decoder (usato da `-trace`/`-disasm`).
- **interrupts.go**: `INT n`/`IRET`/`INTO`, vettori a 0x0000, #DE su div-by-zero;
  API pubblica **`Interrupt(n)`** (sveglia da HLT) e **`InterruptsEnabled()`** —
  usate da retronet-pc per consegnare gli IRQ hardware.
- **memory.go**/**io.go**: bus 1 MB mappabile (`Bus`) e spazio I/O 64K (`Ports`);
  retronet-pc vi innesta le proprie implementazioni assegnando `c.Mem` e `c.IO`.

## Stato

`go test ./...` verde. **SingleStepTests v2 (8088): 99,15%** (2.981.451/3.007.000);
**tutto il comportamento *definito* passa al 100%** (inclusi opcode non documentati
— alias `Jcc` 0x60-0x6F, `SETMO`/`SETMOC`, `SALC` — e quirk come `PUSH SP`). I
~25.500 casi residui sono **comportamento indefinito** del silicio (flag impilati
dopo #DE su DIV/IDIV/AAM; DAA/DAS su BCD non validi), non bug.

Tag: `v0.1.0`, **`v0.1.1`** (API interrupt pubblica). retronet-pc usa `v0.1.0+`.

Prossimi passi: replica opzionale dei flag indefiniti per i casi #DE (resa totale
sulla suite). Il resto del lavoro PC vive in `retronet-pc`.
