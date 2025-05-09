package services

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/helpers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"log/slog"
)

func ExecuteDumpMemory(pid uint, size int, path string) error{
	slog.Info(fmt.Sprintf("## PID: <%d> - Memory Dump solicitado", pid))
	dumpName := helpers.GetDumpName(pid)
	
	err := helpers.CreateFile(models.MemoryConfig.DumpPath + dumpName)
	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return err
	}
	
	return nil
}