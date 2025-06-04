package models

import (
	"sync"
)

// CacheEntry representa una entrada en la caché de páginas.
type CacheEntry struct {
	PID        int // PID del proceso al que pertenece esta página (esencial para desalojo)
	PageNumber int // Identificador único de la página lógica
	//PhysicalAddress int       // Dirección física en memoria (para escribir de vuelta a MP)
	Content     []byte // Contenido de la página (bytes)
	UsageBit    bool   // Bit de Uso (U): true si la página fue accedida recientemente
	ModifiedBit bool   // Bit de Modificación (M): true si la página fue escrita en caché
	//TiempoCarga time.Time // Momento en que la página fue cargada en caché
}

// PageCache representa la caché de páginas de la CPU.
type PageCache struct {
	Entries      []CacheEntry // Slice de punteros a entradas de caché para permitir nil
	MaxEntries   int          // Cantidad máxima de entradas
	Algorithm    string       // Algoritmo de reemplazo (CLOCK o CLOCK-M)
	ClockPointer int          // Puntero para el algoritmo CLOCK/CLOCK-M
	Mutex        sync.Mutex   // Mutex para control de concurrencia
	//PageSize     int          // Tamaño de cada página (para leer de memoria)
	// Mapa para búsqueda rápida: {PID + PageNumber} -> Índice en Entries
	PageMap map[struct {
		PID        int
		PageNumber int
	}]int
}
