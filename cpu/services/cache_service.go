package services

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"log/slog"
	"sync"
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

func InitCache() *PageCache {
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
	cache.Mutex.Lock()
	defer cache.Mutex.Unlock()

	if !IsEnabled(cache.MaxEntries) {
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
