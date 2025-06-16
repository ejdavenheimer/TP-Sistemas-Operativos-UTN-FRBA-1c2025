package services

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
	"log/slog"
	"sync"
	"time"
)

// PageCache representa la caché de páginas de la CPU.
type PageCache struct {
	Entries      []models.CacheEntry // Slice de punteros a entradas de caché para permitir nil
	MaxEntries   int                 // Cantidad máxima de entradas
	Algorithm    string              // Algoritmo de reemplazo (CLOCK o CLOCK-M)
	ClockPointer int                 // Puntero para el algoritmo CLOCK/CLOCK-M
	Mutex        sync.Mutex          // Mutex para control de concurrencia
	//PageSize     int          // Tamaño de cada página (para leer de memoria)
	// Mapa para búsqueda rápida: {PID + PageNumber} -> Índice en Entries
	PageMap map[struct {
		PID        int
		PageNumber int
	}]int
}

var Cache *PageCache

func InitCache() {
	Cache = SetupCache()
}

func SetupCache() *PageCache {
	maxEntries := models.CpuConfig.CacheEntries

	//Debería ser un valor positivo pero si llega a venir negativo lo pongo en 0
	if maxEntries < 0 {
		maxEntries = 0
	}

	cache := &PageCache{
		Entries:      make([]models.CacheEntry, 0, maxEntries),
		MaxEntries:   maxEntries,
		Algorithm:    models.CpuConfig.CacheReplacement,
		ClockPointer: 0,
		PageMap: make(map[struct {
			PID        int
			PageNumber int
		}]int),
	}

	slog.Debug(fmt.Sprintf("Caché de páginas inicializada. MaxEntries: %d, Algoritmo: %s", cache.MaxEntries, cache.Algorithm))
	return cache
}

// IsEnabled verifica si la caché de páginas está habilitada.
func IsEnabled(maxEntries int) bool {
	return maxEntries > 0
}

// getEntryKey genera una clave única para el mapa interno.
func getEntryKey(pid, pageNumber int) struct {
	PID        int
	PageNumber int
} {
	return struct {
		PID        int
		PageNumber int
	}{PID: pid, PageNumber: pageNumber}
}

// Get intenta obtener una página de la caché.
// Retorna el contenido de la página y true si es un caché hit.
func (cache *PageCache) Get(pid, page int) ([]byte, bool) {
	time.Sleep(time.Duration(models.CpuConfig.CacheDelay) * time.Millisecond)

	cache.Mutex.Lock()
	defer cache.Mutex.Unlock()

	if !IsEnabled(cache.MaxEntries) {
		slog.Debug("La cache se encuentra deshabilitada. Operación ignorada.")
		return nil, false
	}

	key := getEntryKey(pid, page)
	index, found := cache.PageMap[key]
	if !found {
		//TODO: revisar que pasa en este caso, entiendo que tendría que revisar si se encuentra en la TLB.
		//Cache MISS
		slog.Debug(fmt.Sprintf("Cache MISS: PID %d, Page %d", pid, page))
		return nil, false
	}

	//Cache HIT
	cache.Entries[index].UseBit = true
	slog.Debug(fmt.Sprintf("Cache HIT: PID %d, Page %d (slot %d). Content: %s", pid, page, index, cache.Entries[index].Content))
	return cache.Entries[index].Content, true
}

// Put añade una página a la caché o actualiza una existente.
func (cache *PageCache) Put(pid, pageNumber int, content []byte) {
	time.Sleep(time.Duration(models.CpuConfig.CacheDelay) * time.Millisecond)

	cache.Mutex.Lock()
	defer cache.Mutex.Unlock()

	if !IsEnabled(cache.MaxEntries) {
		slog.Debug("La cache se encuentra deshabilitada. Operación ignorada.")
		return
	}

	key := getEntryKey(pid, pageNumber)

	// Si ya existe: actualizar
	if index, found := cache.PageMap[key]; found {
		slog.Debug(fmt.Sprintf("Cache Update: PID %d, Page %d (slot %d). Contenido actualizado.", pid, pageNumber, index))
		cache.Entries[index].Content = content
		cache.Entries[index].ModifiedBit = true // Fue modificada en caché
		cache.Entries[index].UseBit = true      // Fue accedida
		return
	}

	//Si no hay espacio, se debe aplicar el algoritmo de sustitución
	if len(cache.Entries) >= cache.MaxEntries {
		cache.replaceVictim(pid, pageNumber, content)
		return
	}

	//Si no existe pero hay espacio libre
	newCacheEntry := models.CacheEntry{
		PID:         pid,
		PageNumber:  pageNumber,
		Content:     content,
		ModifiedBit: true,
		UseBit:      true,
	}
	cache.Entries = append(cache.Entries, newCacheEntry)
	cache.PageMap[key] = len(cache.Entries) - 1
	slog.Debug(fmt.Sprintf("Cache Add: PID %d, Page %d en nuevo slot %d. Total: %d/%d", pid, pageNumber, len(cache.Entries)-1, len(cache.Entries), cache.MaxEntries))
}

func (cache *PageCache) replaceVictim(newPID int, newPage int, newContent []byte) {
	slog.Debug(fmt.Sprintf("Caché llena. Aplicando algoritmo de reemplazo: %s", cache.Algorithm))
	var victimIndex int
	switch cache.Algorithm {
	case "CLOCK":
		victimIndex = cache.findVictimIndexClock()
	case "CLOCK-M":
		victimIndex = cache.findVictimIndexClockM()
	default:
		//TODO: preguntar si puede suceder este caso
		victimIndex = cache.ClockPointer //asigno por default
	}

	victim := cache.Entries[victimIndex]
	slog.Debug(fmt.Sprintf("Víctima seleccionada (slot %d): PID %d, Page %d (U=%t, M=%t)", victimIndex, victim.PID, victim.PageNumber, victim.UseBit, victim.ModifiedBit))

	if victim.ModifiedBit && cache.Algorithm == "CLOCK" {
		slog.Debug(fmt.Sprintf("Víctima (PID %d, Page %d) modificada. Escribiendo a Memoria Principal.", victim.PID, victim.PageNumber))
		err := services.WriteToMemory(uint(victim.PID), victim.PageNumber, newContent)
		if err != nil {
			slog.Error(fmt.Sprintf("WRITE failed, unable to write to memory: %v", err))
			return
		}
	}

	// Eliminar de pageMap antes de reemplazar en Entries
	delete(cache.PageMap, getEntryKey(victim.PID, victim.PageNumber))

	cache.Entries[victimIndex] = models.CacheEntry{
		PID:         newPID,
		PageNumber:  newPage,
		Content:     newContent,
		ModifiedBit: true,
		UseBit:      true,
	}

	cache.PageMap[getEntryKey(victim.PID, victim.PageNumber)] = victimIndex //victim.PageNumber

	slog.Debug(fmt.Sprintf("Nueva página (PID %d, Page %d) cargada en slot %d de caché.", newPID, newPage, victimIndex))
	cache.advancePointer()
}

// findVictimIndexCLOCK implementa el algoritmo CLOCK.
func (cache *PageCache) findVictimIndexClock() int {
	for {
		entry := &cache.Entries[cache.ClockPointer]

		// Si bit de uso es 0, esta es la víctima
		if !entry.UseBit {
			ptr := cache.ClockPointer
			return ptr
		}

		// Si bit de uso es 1, lo reseteo en 0 y avanzo el puntero
		entry.UseBit = false
		cache.advancePointer()
	}
}

// findVictimIndexCLOCK_M implementa el algoritmo CLOCK-M.
func (cache *PageCache) findVictimIndexClockM() int {
	for {
		startIndex := cache.ClockPointer

		//Primer pasada: busca (0,0)
		for i := 0; i < cache.MaxEntries; i++ {
			entry := &cache.Entries[cache.ClockPointer]

			if !entry.UseBit && !entry.ModifiedBit {
				//encontro (0,0)
				ptr := cache.ClockPointer
				return ptr
			}

			// Si es (1,X), poner U=0
			if entry.UseBit {
				entry.UseBit = false
			}

			cache.advancePointer()
		}

		// Segunda pasada: Buscar (0,1)
		// Todas las páginas (1,X) se han convertido en (0,X) en la primera pasada.
		cache.ClockPointer = startIndex
		for i := 0; i < cache.MaxEntries; i++ {
			entry := &cache.Entries[cache.ClockPointer]
			if !entry.UseBit && entry.ModifiedBit {
				//encontro (0,1)
				ptr := cache.ClockPointer
				return ptr
			}
			cache.advancePointer()
		}

		// Si llegamos aquí, no se encontró (0,0) ni (0,1).
		return cache.ClockPointer
	}
}

// advanceHand mueve el puntero del reloj circularmente.
func (cache *PageCache) advancePointer() {
	cache.ClockPointer = (cache.ClockPointer + 1) % cache.MaxEntries
}

// RemoveProcess desalojar todas las páginas de un Proceso específico de la caché.
// Las páginas modificadas se escriben de vuelta a la memoria principal.
func (cache *PageCache) RemoveProcess(pid int) {
	cache.Mutex.Lock()
	defer cache.Mutex.Unlock()

	if !IsEnabled(cache.MaxEntries) {
		slog.Error(fmt.Sprintf("Se intento de desalojar proceso %d de caché deshabilitada. Operación ignorada.", pid))
		return
	}

	slog.Debug(fmt.Sprintf("Desalojando páginas del Proceso %d de la caché.", pid))

	newEntries := make([]models.CacheEntry, 0, cache.MaxEntries)
	newMap := make(map[struct {
		PID        int
		PageNumber int
	}]int)

	for _, entry := range cache.Entries {
		if entry.PID != pid {
			// Mantener esta entrada y añadirla al nuevo slice y mapa
			newEntries = append(newEntries, entry)
			newMap[getEntryKey(entry.PID, entry.PageNumber)] = len(newEntries) - 1 // Actualizar índice en el nuevo mapa
			continue
		}

		slog.Debug(fmt.Sprintf("DESALOJO: Encontrada página %d del Proceso %d. U=%t, M=%t.", entry.PageNumber, pid, entry.UseBit, entry.ModifiedBit))
		if entry.ModifiedBit {
			slog.Debug(fmt.Sprintf("DESALOJO: Página %d (Proceso %d) modificada. Escribiendo a Memoria Principal.", entry.PageNumber, pid))
			err := services.WriteToMemory(uint(pid), entry.PageNumber, entry.Content)
			if err != nil {
				slog.Error(fmt.Sprintf("WRITE failed, unable to write to memory: %v", err))
				return
			}
		}
	}

	cache.Entries = newEntries
	cache.PageMap = newMap

	// Reajustar ClockHand: Si el tamaño de Entries cambió, el ClockHand podría estar fuera de rango
	if cache.MaxEntries > 0 && len(cache.Entries) > 0 {
		cache.ClockPointer = cache.ClockPointer % len(cache.Entries) // Asegura que esté dentro de los límites del nuevo tamaño
	} else { // Si la caché está vacía o deshabilitada
		cache.ClockPointer = 0
	}

	slog.Debug(fmt.Sprintf("Páginas del Proceso %d desalojadas. Entradas restantes en caché: %d", pid, len(cache.Entries)))
}
