package config

import (
	"encoding/json"
	"fmt"
	"os"
    "path/filepath"
)

// InitConfig lee el archivo de configuración y retorna sus valores en la variable config. En caso de error no se crea el archivo
//
// Parámetros:
//   - filePath: ubicacion donde se encuentra el archivo de configuracion
//   - config: acepta cualquier tipo de estructura
//
// Ejemplo:
//
//	type TestConfig struct {
//		Name  string `json:"name"`
//		Value int    `json:"value"`
//	}
//	func main() {
//		var testConfig TestConfig
//		config.InitConfig("./test.json", &testConfig)
//	}
func InitConfig(filePath string, config interface{}) {
	err := setupConfig(filePath, &config)
	if err != nil {
		_ = fmt.Errorf("error al configurar el archivo %v", err)
		panic(err)
	}
}

func setupConfig(filePath string, config interface{}) error {
	configFile, err := os.Open(filePath)

	if err != nil {
		return err
	}

	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)

	if err := jsonParser.Decode(&config); err != nil {
		return err
	}

	return nil
}

func GetProjectRoot() string {
    exePath, err := os.Executable()
    if err != nil {
        panic("no se pudo obtener el path del ejecutable: " + err.Error())
    }

    exeDir := filepath.Dir(exePath)
    return filepath.Join(exeDir, "..") // asume que binarios están en /bin/
}

// Paths de config.json para cada módulo
func KernelConfigPath() string { 
	return filepath.Join(GetProjectRoot(), "kernel/configs/kernel.json")
}

func MemoriaConfigPath() string {
    return filepath.Join(GetProjectRoot(), "memoria/configs/memoria.json")
}

func CpuConfigPath() string {
    return filepath.Join(GetProjectRoot(), "cpu/configs/cpu.json")
}

func IOConfigPath() string {
    return filepath.Join(GetProjectRoot(), "io/configs/io.json")
}

// Paths de log para cada módulo
func KernelLogPath() string {
    return filepath.Join(GetProjectRoot(), "logs/kernel.log")
}

func MemoriaLogPath() string {
    return filepath.Join(GetProjectRoot(), "logs/memoria.log")
}