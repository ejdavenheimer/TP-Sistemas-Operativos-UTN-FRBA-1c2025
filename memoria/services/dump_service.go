package services

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/helpers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"log/slog"
)

func ExecuteDumpMemory(pid uint, size int, path string) error {
	slog.Info(fmt.Sprintf("## PID: %d - Memory Dump solicitado", pid)) //Log obligatorio
	dumpName := helpers.GetDumpName(pid)                               //obtiene el nombre del archivo

	//crea o abre el archivo
	dumpFilePath := models.MemoryConfig.DumpPath + dumpName
	file, err := helpers.CreateFile(dumpFilePath, size)
	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return err
	}

	dumpContent := make([]byte, size)
	numberPages := size / models.MemoryConfig.PageSize

	if size%models.MemoryConfig.PageSize != 0 {
		numberPages++ // Si el tamaño no es múltiplo de PageSize, redondear hacia arriba
	}

	for pageNum := 0; pageNum < numberPages; pageNum++ {
		pageContent, err := GetPageContent(pid, pageNum) // Asumo este método en MemoriaService
		if err != nil {
			slog.Error(fmt.Sprintf("Memoria: Fallo al leer página %d para PID %d durante el dump", pageNum, pid))
			// Decidir si continuar o abortar el dump. Por simplicidad, abortamos aquí.
			return fmt.Errorf("fallo al leer página %d durante el dump: %w", pageNum, err)
		}
		dumpContent = append(dumpContent, pageContent...)
	}

	// 5. Escribir el contenido en el archivo
	_, err = file.Write(dumpContent[:size]) // Escribir solo hasta el 'size' real
	if err != nil {
		slog.Error(fmt.Sprintf("Memoria: Fallo al escribir contenido en el archivo de dump '%s'", dumpFilePath))
		return fmt.Errorf("fallo al escribir datos al archivo de dump: %w", err)
	}

	slog.Info(fmt.Sprintf("Memoria: Memory Dump completado para PID %d. Archivo: %s", pid, dumpFilePath))

	return nil
}

func GetPageContent(pid uint, pageNum int) ([]byte, error) {
	slog.Warn("TODO: implementar ")
	return nil, nil
}
