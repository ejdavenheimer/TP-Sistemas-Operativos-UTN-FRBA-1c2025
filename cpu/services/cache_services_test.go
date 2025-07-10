package services

import (
	"testing"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
)

func TestMain(m *testing.M) {
	// Configuración que se ejecuta una única vez antes de todos los tests
	config.InitConfig("../../cpu/configs/cpu.json", &models.CpuConfig)
	Cache = SetupCache()

	if Cache == nil {
		panic("Expected cache to not be nil in TestMain") // Usa panic para fallos críticos en la configuración
	}

	m.Run()
}

func TestCacheMiss(t *testing.T) {
	//services.Cache := testCache
	Cache.MaxEntries = 0

	result, found := Cache.Get(1, 10)
	if found || result != nil {
		t.Fatalf("Expected cache to not be found")
	}

	Cache.MaxEntries = 3
	result, found = Cache.Get(1, 10)
	if found || result != nil {
		t.Fatalf("Expected cache to not be found")
	}
}

func TestAlgorithmClock(t *testing.T) {
	//Cache := testCache
	Cache.Algorithm = "CLOCK"
	Cache.MaxEntries = 3

	services.WriteToMemoryMock = func(pid uint, page int, content []byte) error {
		return nil
	}
	defer func() { services.WriteToMemoryMock = services.WriteToMemory }()

	Cache.Put(0, 10, 0, []byte("test 0"))
	result, found := Cache.Get(0, 10)

	if !found {
		t.Errorf("Expected cache to not be found")
	}

	if !found || string(result) != "test 0" {
		t.Errorf("Expected cache to be 'test 0', got %s", string(result))
	}
	Cache.Put(1, 20, 0, []byte("test 1"))
	result, found = Cache.Get(0, 100)
	if found || result != nil {
		t.Errorf("Expected cache to not be found")
	}

	result, found = Cache.Get(1, 20)
	if !found || string(result) != "test 1" {
		t.Errorf("Expected cache to be 'test 1', got %s", string(result))
	}

	Cache.Put(1, 20, 0, []byte("test 2"))
	Cache.Put(3, 30, 0, []byte("test 3"))
	Cache.Put(4, 40, 0, []byte("test 4"))
	Cache.Put(5, 50, 0, []byte("test 5"))
}

// Se prueba con (0,1)
func TestAlgorithmClockM(t *testing.T) {
	//Cache := testCache
	Cache.Algorithm = "CLOCK-M"
	Cache.MaxEntries = 3

	services.WriteToMemoryMock = func(pid uint, page int, content []byte) error {
		return nil
	}
	defer func() { services.WriteToMemoryMock = services.WriteToMemory }()

	Cache.Put(0, 10, 0, []byte("test 0"))
	Cache.Put(1, 20, 0, []byte("test 1"))
	Cache.Put(1, 20, 0, []byte("test 2"))
	Cache.Put(3, 30, 0, []byte("test 3"))
	Cache.Put(4, 40, 0, []byte("test 4"))
	Cache.Put(5, 50, 0, []byte("test 5"))
}

func TestPageCache_RemoveProcess(t *testing.T) {
	//Cache := testCache
	Cache.MaxEntries = 3

	Cache.Put(0, 10, 0, []byte("test 0"))
	Cache.Put(1, 20, 0, []byte("test 1"))
	Cache.Put(2, 30, 0, []byte("test 2"))

	Cache.RemoveProcessFromCache(1)
}
