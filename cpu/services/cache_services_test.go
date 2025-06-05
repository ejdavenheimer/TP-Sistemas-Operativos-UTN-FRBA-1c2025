package services

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"testing"
)

func TestInitCache(t *testing.T) {
	config.InitConfig("../../cpu/configs/cpu.json", &models.CpuConfig)
	var cache *PageCache = InitCache()

	if cache == nil {
		t.Errorf("Expected cache to not be nil")
	}

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

	cache.Put(2, 20, []byte("test 2"))
	cache.Put(3, 30, []byte("test 3"))
	cache.Put(4, 40, []byte("test 4"))
	cache.Put(5, 50, []byte("test 5"))

}
