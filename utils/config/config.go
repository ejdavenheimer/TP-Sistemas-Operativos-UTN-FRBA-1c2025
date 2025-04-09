package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func InitConfig(filePath string, config interface{}) {
	err := setupConfig(filePath, &config)
	if err != nil {
		fmt.Errorf("Error al configurar el archivo: %v", err)
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
