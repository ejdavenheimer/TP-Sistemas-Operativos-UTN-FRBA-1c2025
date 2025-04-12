package config

import (
	"encoding/json"
	"fmt"
	"os"
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
