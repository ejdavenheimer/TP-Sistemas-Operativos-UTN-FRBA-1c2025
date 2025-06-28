package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	_ "strconv"
)

// Para su uso se debe posicionar en la carpeta scripts
// > ./update_config.exe ip_memory 192.168.1.100
// > ./update_config.exe ip_memory 192.168.1.100 ip_cpu 192.168.1.101 ip_kernel 192.168.1.102 ip_io 192.168.1.103
// > ./update_config.exe ip_memory 127.0.0.1 ip_cpu 127.0.0.1 ip_kernel 127.0.0.1 ip_io 127.0.0.1

func main() {
	// Verificar que se pasen argumentos en pares: clave1 valor1 clave2 valor2 ...
	if len(os.Args) < 3 || len(os.Args)%2 != 1 { // Debe ser al menos 3 (programa, clave, valor) y un número impar de argumentos totales
		fmt.Println("Uso: update_config <clave_ip_1> <valor_ip_1> [<clave_ip_2> <valor_ip_2> ...]")
		fmt.Println("Ejemplo: update_config ip_memory 192.168.0.10 ip_kernel 192.168.0.20")
		return
	}

	// Mapear los argumentos de línea de comandos a un mapa de claves a sus nuevos valores
	updates := make(map[string]interface{})
	for i := 1; i < len(os.Args); i += 2 {
		key := os.Args[i]
		valueStr := os.Args[i+1]

		// Intentar unmarshal el valor a su tipo real (ej. números, booleanos, etc.)
		// Si falla, se trata como string. Esto es más robusto para otros valores,
		// aunque para IPs siempre serán strings. Lo mantengo por si la función evoluciona.
		var parsedValue interface{}
		err := json.Unmarshal([]byte(valueStr), &parsedValue)
		if err != nil {
			// Si no es un JSON válido (ej. un string simple como una IP), usar el string directamente
			parsedValue = valueStr
		}
		updates[key] = parsedValue
	}

	fmt.Println("Valores a actualizar:")
	for k, v := range updates {
		fmt.Printf("  %s: %v\n", k, v)
	}

	// Definimos las carpetas de los módulos que queremos procesar.
	modules := []string{"cpu", "io", "kernel", "memoria"}

	// Iteramos sobre cada módulo
	for _, module := range modules {
		moduleConfigPath := filepath.Join("..", module, "configs")
		fmt.Printf("\nProcesando módulo: %s (en %s)\n", module, moduleConfigPath)

		err := filepath.Walk(moduleConfigPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("  Error al acceder %s: %v\n", path, err)
				return nil
			}
			if !info.IsDir() && filepath.Ext(path) == ".json" {
				fileContent, err := os.ReadFile(path)
				if err != nil {
					fmt.Printf("  Error al leer el archivo %s: %v\n", path, err)
					return nil
				}

				var data map[string]interface{}
				err = json.Unmarshal(fileContent, &data)
				if err != nil {
					fmt.Printf("  Error al parsear JSON en el archivo %s: %v\n", path, err)
					return nil
				}

				modified := false

				// Iteramos sobre las claves y valores que el usuario quiere actualizar
				for updateKey, updateValue := range updates {
					// Verificamos si la clave de actualización existe en el JSON actual
					if _, ok := data[updateKey]; ok {
						data[updateKey] = updateValue
						fmt.Printf("    Modificada '%s' en %s a '%v'\n", updateKey, path, updateValue)
						modified = true
					}
				}

				// Si se realizaron modificaciones, sobrescribimos el archivo
				if modified {
					newJSON, err := json.MarshalIndent(data, "", "  ")
					if err != nil {
						fmt.Printf("  Error al serializar JSON en el archivo %s: %v\n", path, err)
						return nil
					}

					err = os.WriteFile(path, newJSON, 0644)
					if err != nil {
						fmt.Printf("  Error al escribir el archivo %s: %v\n", path, err)
						return nil
					}
					fmt.Printf("  El archivo %s ha sido actualizado correctamente.\n", path)
				} else {
					fmt.Printf("  No se encontraron claves a actualizar del tipo IP en %s.\n", path)
				}
			}
			return nil
		})

		if err != nil {
			fmt.Printf("Error al buscar archivos en la carpeta %s: %v\n", moduleConfigPath, err)
		}
	}

	fmt.Println("\nProceso de actualización de configuraciones finalizado.")
}
