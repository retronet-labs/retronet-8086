# retronet-8086 — Emulatore Intel 8086/8088

Emulatore della CPU Intel 8086/8088 in real mode, scritto in Go, parte
dell'ecosistema **RetroNet**. Segue la filosofia dei moduli 4004/8008/8080: core
importabile, due ALU intercambiabili (a **porte logiche** e **native**),
test e documentazione in italiano.

È il motore del futuro **retronet-pc** (un IBM PC/XT compatibile): per questo il
profilo di default è l'**8088** e la memoria è già segmentata su 1 MB.

## Le due ALU (gate e native)

Come gli altri core RetroNet, l'aritmetica è delegata a un backend
intercambiabile, identico nei risultati e nei flag (garanzia del test
differenziale `TestGateVsNativeALUDifferential`):

- `cpu.Gate` (default) — l'ALU costruita dalle sole **porte logiche** di
  `retronet-logic`, raggiunta dal bridge
  [`retronet-hardware/bridge/i8086`](https://github.com/retronet-labs/retronet-hardware).
  Anche **moltiplicazione** (shift-and-add) e **divisione** (a ripristino) sono
  composte sul sommatore a gate.
- `cpu.Native` — la stessa semantica con gli operatori di Go: più veloce, fa da
  oracolo del differenziale.

```go
c := cpu.NewCPU8086()        // ALU a porte (default), profilo 8088
c.SetALU(cpu.Native)         // oppure: cpu.NewCPU8086WithALU(cpu.Native)
```

## Profili 8086 / 8088

ISA, risultati e flag sono identici; cambiano solo bus dati esterno e coda di
prefetch (e quindi il timing):

```go
c.Profile = cpu.Profile8086 // 16 bit, coda 6 byte
c.Profile = cpu.Profile8088 // 8 bit,  coda 4 byte (default, IBM XT)
```

## CLI

```bash
go run ./cmd/retronet-8086 -profiles                 # elenca 8086/8088
go run ./cmd/retronet-8086 -conformance              # batteria sintetica
go run ./cmd/retronet-8086 -bin prog.bin -trace      # esegue un binario raw
go run ./cmd/retronet-8086 -bin prog.bin -alu native # con ALU native
go run ./cmd/retronet-8086 -testsuite <dir-vettori>  # SingleStepTests 8088
```

## Esempio

```go
c := cpu.NewCPU8086()
c.Seg[cpu.CS], c.IP = 0x0000, 0x0100
c.Mem.(*cpu.RAM).LoadAt(cpu.PhysAddr(0x0000, 0x0100), []byte{
    0xB8, 0x00, 0x00, // MOV AX,0
    0xB9, 0x05, 0x00, // MOV CX,5
    0x01, 0xC8, // ADD AX,CX
    0xE2, 0xFC, // LOOP -4
    0xF4, // HLT
})
c.Run(1000)        // AX = 15, calcolato sull'ALU a porte
```

## Stato

Implementato e testato (`go test ./...` verde):

- Registri (AX..DI con mezzi-registri AL/AH...), segmenti, FLAGS a 16 bit con i
  bit riservati dell'8086, indirizzamento fisico a 20 bit con wrap a 1 MB.
- Backend ALU **Gate**/**Native** con flag CF/PF/AF/ZF/SF/OF, INC/DEC,
  MUL/IMUL, DIV/IDIV — differenziale esaustivo a 8 bit, campionato a 16 bit.
- Decodifica ModR/M (indirizzamento a 16 bit, override di segmento) e famiglie:
  MOV (tutte le forme), blocco aritmetico-logico completo (`00`-`3D` e gruppo
  `80`-`83`), INC/DEC, PUSH/POP (registri, segmenti, `PUSHF`/`POPF`), XCHG,
  TEST, NOT/NEG/MUL/DIV (`F6`/`F7`), shift/rotate (`D0`-`D3`), istruzioni stringa
  `MOVS/STOS/LODS/SCAS/CMPS` con `REP`/`REPNE`, `LDS`/`LES`/`XLAT`, aggiustamenti
  BCD/ASCII (`DAA/DAS/AAA/AAS/AAM/AAD`), salti e `Jcc`, `LOOP`/`JCXZ`,
  CALL/RET(F), INT/IRET/INTO (con #DE su divisione per zero), operazioni sui
  flag, CBW/CWD, SAHF/LAHF, IN/OUT, HLT, prefissi (override di segmento, LOCK, REP).
- CLI `cmd/retronet-8086` (esecuzione raw, profili, conformance, testsuite),
  batteria di **conformance** sintetica e loader **SingleStepTests** (TomHarte)
  per la validazione per-istruzione (dataset fuori dal repo). Vedi
  [docs/architettura.md](docs/architettura.md).

## Validazione SingleStepTests (TomHarte)

Sull'**intero set v2** dei vettori per-istruzione 8088 (323 file, 3.007.000 casi)
il core supera **99,15%** (2.981.451/3.007.000), con i flag indefiniti mascherati
per-opcode. **Tutte le istruzioni a comportamento definito passano al 100%** —
l'intera ISA dell'8088, compresi gli opcode non documentati (alias `Jcc` 0x60-0x6F,
`SETMO`/`SETMOC`, `SALC`) e i quirk noti (`PUSH SP`).

I ~25.500 casi residui sono **comportamento indefinito** dell'8086, non bug:

- `DIV`/`IDIV`/`AAM` in errore di divisione (#DE) impilano sullo stack i flag che
  il silicio lascia indefiniti dopo l'operazione abortita (non mascherabili a
  livello di byte di RAM);
- `DAA`/`DAS` su input BCD **non validi** (nibble fuori 0-9), che producono
  risultati specifici del silicio.

Il loader distingue *passati* / *errati* / *non implementati*. Per riprodurre:
scaricare i file `v2/*.json.gz` da
[SingleStepTests/8088](https://github.com/SingleStepTests/8088) in una cartella e

```bash
go run ./cmd/retronet-8086 -testsuite <dir-vettori>   # oppure -alu native (piu' veloce)
```

In lavorazione (prossimi passi):

- Disassembler simmetrico al decoder (per un trace leggibile).
- Replica opzionale dei flag indefiniti del silicio per i casi #DE (per la resa
  totale su SingleStepTests).

## Sviluppo locale (multi-repo)

Dipende da `retronet-hardware` (bridge `i8086`, da `v0.7.1`) e `retronet-logic`.
Un clone pulito compila dalle versioni pubblicate; per co-sviluppare in locale si
usano i checkout sibling con un `go.work` (non versionato):

```sh
# clona retronet-logic, retronet-hardware e retronet-8086 come cartelle sibling
go work init . ../retronet-hardware ../retronet-logic   # già presente nel repo
go test ./...
```

## Licenza

MIT.
