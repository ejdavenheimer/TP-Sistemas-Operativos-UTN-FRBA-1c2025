package services

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"log/slog"
)

func InitCache() *models.PageCache {
	maxEntries := models.CpuConfig.CacheEntries

	//Debería ser un valor positivo pero si llega a venir negativo lo pongo en 0
	if maxEntries < 0 {
		maxEntries = 0
	}

	cache := &models.PageCache{
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
