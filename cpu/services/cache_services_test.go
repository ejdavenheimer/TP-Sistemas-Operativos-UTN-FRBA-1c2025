package services

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"testing"
)

var testCache *PageCache

func TestMain(m *testing.M) {
	// Configuración que se ejecuta una única vez antes de todos los tests
	config.InitConfig("../../cpu/configs/cpu.json", &models.CpuConfig)
	testCache = SetupCache()

	if testCache == nil {
		panic("Expected cache to not be nil in TestMain") // Usa panic para fallos críticos en la configuración
	}

	m.Run()
}

func TestCacheMiss(t *testing.T) {
	cache := testCache
	cache.MaxEntries = 0

	result, found := cache.Get(1, 10)
	if found || result != nil {
		t.Fatalf("Expected cache to not be found")
	}

	cache.MaxEntries = 3
	result, found = cache.Get(1, 10)
	if found || result != nil {
		t.Fatalf("Expected cache to not be found")
	}
}

func TestAlgorithmClock(t *testing.T) {
	cache := testCache
	cache.Algorithm = "CLOCK"
	cache.MaxEntries = 3

	cache.Put(0, 10, []byte("test 0"))
	result, found := cache.Get(0, 10)

	if !found {
		t.Errorf("Expected cache to not be found")
	}

	if !found || string(result) != "test 0" {
		t.Errorf("Expected cache to be 'test 0', got %s", string(result))
	}
	cache.Put(1, 20, []byte("test 1"))
	result, found = cache.Get(0, 100)
	if found || result != nil {
		t.Errorf("Expected cache to not be found")
	}

	result, found = cache.Get(1, 20)
	if !found || string(result) != "test 1" {
		t.Errorf("Expected cache to be 'test 1', got %s", string(result))
	}

	cache.Put(1, 20, []byte("test 2"))
	cache.Put(3, 30, []byte("test 3"))
	cache.Put(4, 40, []byte("test 4"))
	cache.Put(5, 50, []byte("test 5"))
}

// se prueba con (0,1)
func TestAlgorithmClockM(t *testing.T) {
	cache := testCache
	cache.Algorithm = "CLOCK-M"
	cache.MaxEntries = 3

	cache.Put(0, 10, []byte("test 0"))
	cache.Put(1, 20, []byte("test 1"))
	cache.Put(1, 20, []byte("test 2"))
	cache.Put(3, 30, []byte("test 3"))
	cache.Put(4, 40, []byte("test 4"))
	cache.Put(5, 50, []byte("test 5"))
}

func TestPageCache_RemoveProcess(t *testing.T) {
	cache := testCache
	cache.MaxEntries = 3

	cache.Put(0, 10, []byte("test 0"))
	cache.Put(1, 20, []byte("test 1"))
	cache.Put(2, 30, []byte("test 2"))

	cache.RemoveProcess(1)
}
