package models

// CacheEntry representa una entrada en la caché de páginas.
type CacheEntry struct {
	PID        uint // PID del proceso al que pertenece esta página (esencial para desalojo)
	PageNumber int // Identificador único de la página lógica
	//PhysicalAddress int       // Dirección física en memoria (para escribir de vuelta a MP)
	Content     []byte // Contenido de la página (bytes)
	UseBit      bool   // Bit de Uso (U): true si la página fue accedida recientemente
	ModifiedBit bool   // Bit de Modificación (M): true si la página fue escrita en caché
	LockerBit   bool   // Bit de bloqueo: true esta siendo leida o escrita por lo que no puede ser reemplazada
	//TiempoCarga time.Time // Momento en que la página fue cargada en caché
}
