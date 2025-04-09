package config

import (
	"encoding/json"
	"os"
	"testing"
)

type TestConfig struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestSetupConfig(t *testing.T) {
	tempFile, err := os.CreateTemp("", "testconfig")

	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}

	defer os.Remove(tempFile.Name())

	validConfig := TestConfig{Name: "test", Value: 123}
	json.NewEncoder(tempFile).Encode(validConfig)
	tempFile.Seek(0, 0) // Reset file pointer

	var config TestConfig
	err = setupConfig(tempFile.Name(), &config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if config != validConfig {
		t.Errorf("Expected config to be %v, got: %v", validConfig, config)
	}
}

func TestSetupConfig_ThrowError(t *testing.T) {
	err := setupConfig("nonexistent.json", &TestConfig{})
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
