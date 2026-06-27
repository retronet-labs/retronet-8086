package cpu

// Profile descrive la variante di chip. L'8086 e l'8088 condividono esattamente
// la stessa ISA e gli stessi risultati: differiscono per la larghezza del bus
// dati esterno e per la dimensione della coda di prefetch (e quindi per il
// timing). La semantica delle istruzioni e dei flag e' identica.
type Profile struct {
	Name          string
	PrefetchBytes int // coda di prefetch: 6 sull'8086, 4 sull'8088
	DataBusBits   int // bus dati esterno: 16 sull'8086, 8 sull'8088
}

// Profili predefiniti.
var (
	Profile8086 = Profile{Name: "8086", PrefetchBytes: 6, DataBusBits: 16}
	Profile8088 = Profile{Name: "8088", PrefetchBytes: 4, DataBusBits: 8}
)

// Profiles elenca i profili disponibili. Il default del core e' l'8088, coerente
// con l'obiettivo IBM PC/XT.
func Profiles() []Profile {
	return []Profile{Profile8088, Profile8086}
}
