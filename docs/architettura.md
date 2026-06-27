# Architettura di retronet-8086

Questo documento spiega come è fatto il core e perché, nello spirito didattico di
RetroNet: capire l'8086 ricostruendolo, con l'aritmetica che gira **davvero** su
porte logiche.

## Le due ALU: gate e native

Come gli altri chip RetroNet, l'8086 non calcola l'aritmetica con gli operatori
di Go: la **delega** a un backend (`cpu.ALUBackend`). Ne esistono due, con
risultati e flag identici:

- **Gate** (default) — inoltra al bridge
  `retronet-hardware/bridge/i8086`, che adatta la `alu` a porte logiche di
  `retronet-logic` alla semantica dell'8086. Anche moltiplicazione e divisione
  sono composte sul sommatore a gate (shift-and-add e divisione a ripristino),
  e shift/rotate iterano lo `shifter` a 1 bit.
- **Native** — la stessa semantica con gli operatori di Go; fa da **oracolo**.

Il test `TestGateVsNativeALUDifferential` confronta i due backend su tutti gli
ingressi a 8 bit e su un campione a 16, per ogni operazione e flag. È la rete che
garantisce che "calcolare coi gate" dia esattamente lo stesso risultato.

## Registri e flag

I registri generali `AX..DI` vivono in un array indicizzato; i mezzi-registri
`AL/AH..` condividono lo stesso storage (vedi `Get8`/`Set8`). I flag sono campi
booleani distinti e si impacchettano nel registro `FLAGS` a 16 bit solo quando
serve (`PUSHF`, `IRET`, confronto coi vettori di test): `PackFlags` applica i bit
riservati che l'8086 espone sempre a 1.

I flag aritmetici (CF/PF/AF/ZF/SF/OF) arrivano dal backend ALU; quelli di
controllo (TF/IF/DF) li gestisce la CPU.

## Segmentazione

L'8086 vede 1 MB con indirizzi a 20 bit. L'indirizzo fisico è
`(segment << 4) + offset`, con wrap a `0xFFFFF` (`PhysAddr`). Ogni accesso usa un
segmento di default (DS per i dati, SS quando l'indirizzamento usa BP, CS per il
fetch, ES per la destinazione delle stringhe), sostituibile da un prefisso di
override. Gli accessi a 16 bit fanno wrap dell'offset a 16 bit nello stesso
segmento, come sull'hardware reale.

## Decodifica: prefissi, ModR/M, gruppi

`Step` raccoglie i prefissi (override di segmento, LOCK, REP/REPNE) e poi
`execute` smista l'opcode. Il blocco aritmetico-logico `0x00-0x3D` è gestito in
modo tabellare (gruppo = `op>>3`, forma = `op&7`), mentre i gruppi a opcode
esteso (`0x80-0x83`, `0xF6/0xF7`, `0xFE/0xFF`, `0xD0-0xD3`) usano il campo *reg*
del byte ModR/M come selettore.

Il ModR/M (`decodeModRM`) calcola, per gli operandi in memoria, la coppia
segmento:offset effettiva secondo le otto combinazioni base/indice
dell'indirizzamento a 16 bit (niente SIB sull'8086), leggendo l'eventuale
displacement dal flusso d'istruzioni.

## Profili 8086 / 8088

`Profile8086` e `Profile8088` hanno ISA, risultati e flag identici: cambiano solo
la larghezza del bus dati esterno e la coda di prefetch (e quindi il timing).
Il default è l'8088, coerente con l'obiettivo IBM PC/XT.

## Validazione

Tre reti complementari:

1. **Differenziale Gate vs Native** sull'ALU (esaustivo a 8 bit).
2. **Conformance sintetica** (`conformance`): programmi auto-verificanti senza
   dati esterni, sempre verdi in CI; raggiungibile da `-conformance`.
3. **SingleStepTests (TomHarte)** (`testsuite`): vettori per-istruzione che
   descrivono CPU+RAM prima/dopo. I dataset stanno fuori dal repo; si passano con
   `-testsuite <dir>` o via `RETRONET_8088_TESTS` nei test.

## Verso retronet-pc

Il core è disegnato per diventare il motore di **retronet-pc** (IBM PC/XT): la
memoria è dietro l'interfaccia `Bus` e l'I/O dietro `Ports`, così retronet-pc
potrà sostituirvi un bus mappato (RAM + video RAM + ROM del BIOS) e le periferiche
(8259/8253/8237/8255, video, floppy) senza toccare la CPU.
