package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

var (
	tlb            []models.TLBEntry
	tlbMaxSize     int
	tlbAlgorithm   string // "FIFO" o "LRU"
	tlbCounter     int64  // para LRU, contador incremental
	tlbFifoPointer int    // **NUEVO**: Puntero para FIFO
	tlbMutex       sync.Mutex
)

func InitTLB() {
	tlbMaxSize = models.CpuConfig.TlbEntries
	tlbAlgorithm = models.CpuConfig.TlbReplacement // "FIFO" o "LRU"
	tlbCounter = 0
	tlbFifoPointer = 0 // **NUEVO**: Inicializamos el puntero
	tlb = make([]models.TLBEntry, 0, tlbMaxSize)
}

func RequestMemoryConfig() error {
	resp, err := client.DoRequest(models.CpuConfig.PortMemory, models.CpuConfig.IpMemory, "GET", "config/memoria")
	if err != nil {
		slog.Error("Error solicitando configuración de Memoria")
		return err
	}

	var config models.MemoryConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		slog.Error("Error decodificando configuración de Memoria")
		return err
	}

	models.MemConfig = &config
	slog.Debug("MemConfig cargada", slog.Any("config", models.MemConfig))
	return nil
}

func TranslateAddress(pid uint, logicalAddress int) int {
	pageSize := models.MemConfig.PageSize
	pageNumber := logicalAddress / pageSize
	offset := logicalAddress % pageSize

	slog.Debug("Traducción de dirección", "pid", pid, "logical", logicalAddress, "pageNumber", pageNumber, "pageSize", pageSize)

	//Verifica que la tlb no este desactivada
	if tlbMaxSize > 0 {
		if frame, ok := searchTLB(pid, pageNumber); ok {
			//si la encuentra imprime TLB HIT y traduce
			slog.Info(fmt.Sprintf("PID: %d - TLB HIT - Pagina: %d", pid, pageNumber))
			slog.Info(fmt.Sprintf("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", pid, pageNumber, frame))
			return frame*pageSize + offset
		}
		slog.Info(fmt.Sprintf("PID: %d - TLB MISS - Página: %d", pid, pageNumber))
		frame := tlb_miss(pid, pageNumber)
		if frame == -1 {
			slog.Warn("Violación de memoria detectada (TLB MISS)", "pid", pid, "page", pageNumber)
			return -1
		}
		insert_tlb(pid, pageNumber, frame)
		slog.Info(fmt.Sprintf("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", pid, pageNumber, frame))
		return frame*pageSize + offset
	}
	// TLB desactivada
	slog.Info(fmt.Sprintf("PID: %d - TLB desactivada - Traducción completa - Pagina: %d", pid, pageNumber))
	frame := tlb_miss(pid, pageNumber)
	if frame == -1 {
		slog.Warn("Violación de memoria detectada (TLB MISS)", "pid", pid, "page", pageNumber)
		return -1
	}
	//slog.Debug("RequestMemoryFrame", "pid", pid, "frame", frame)
	slog.Info(fmt.Sprintf("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", pid, pageNumber, frame))
	return frame*pageSize + offset

}

func tlb_miss(pid uint, pageNumber int) int {
	//slog.Debug("tlb_miss: pid", string(pid), "pageNumber", pageNumber)
	return RequestMemoryFrame(pid, pageNumber)
}

func searchTLB(pid uint, pagina int) (int, bool) {
	tlbMutex.Lock()
	defer tlbMutex.Unlock()

	for i := range tlb {
		if tlb[i].PID == pid && tlb[i].PageNumber == pagina {
			if tlbAlgorithm == "LRU" {
				tlbCounter++
				tlb[i].LastUsed = tlbCounter
			}
			return tlb[i].FrameNumber, true
		}
	}
	return 0, false
}

func insert_tlb(pid uint, pagina int, frame int) {
	tlbMutex.Lock()
	defer tlbMutex.Unlock()

	tlbCounter++
	entry := models.TLBEntry{
		PID:         pid,
		PageNumber:  pagina,
		FrameNumber: frame,
		LastUsed:    tlbCounter,
	}

	if len(tlb) < tlbMaxSize {
		tlb = append(tlb, entry)
		return
	}

	var victimIndex int
	switch tlbAlgorithm {
	case "FIFO":
		victimIndex = tlbFifoPointer
		tlbFifoPointer = (tlbFifoPointer + 1) % tlbMaxSize // Avanza el puntero circularmente
	case "LRU":
		minUsage := tlb[0].LastUsed
		victimIndex = 0
		for i, e := range tlb {
			if e.LastUsed < minUsage {
				minUsage = e.LastUsed
				victimIndex = i
			}
		}
	default:
		slog.Warn("Algoritmo TLB desconocido. Se usará FIFO por defecto.")
		victimIndex = 0
	}

	tlb[victimIndex] = entry
	slog.Debug(fmt.Sprintf("TLB reemplazo: Reemplazando entrada PID %d - Página %d por PID %d - Página %d",
		tlb[victimIndex].PID, tlb[victimIndex].PageNumber,
		entry.PID, entry.PageNumber))
}

func RequestMemoryFrame(pid uint, pageNumber int) int {
	slog.Debug(fmt.Sprintf("RequestMemoryFrame llamado - PID: %d, Página: %d", pid, pageNumber))
	type Request struct {
		PID        uint `json:"pid"`
		PageNumber int  `json:"pageNumber"`
	}
	type Response struct {
		Frame int `json:"frame"`
	}

	reqBody := Request{PID: pid, PageNumber: pageNumber}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		slog.Error("Error serializando JSON para Memoria por frame", slog.Any("error", err))
		return -1
	}

	resp, err := client.DoRequest(models.CpuConfig.PortMemory, models.CpuConfig.IpMemory, "POST", "memoria/buscarFrame", jsonBody)
	if err != nil {
		slog.Error("Error al hacer request a Memoria por frame", slog.Any("error", err))
		return -1
	}
	defer resp.Body.Close()

	var decoded Response
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		slog.Error("Error decodificando respuesta de Memoria", slog.Any("error", err))
		return -1
	}
	//slog.Debug(fmt.Sprintf("RequestMemoryFrame: PID %d Página %d - Frame devuelto %d", pid, pageNumber, decoded.Frame))
	return decoded.Frame
}

// Elimina las TLBs de los procesos que sean finalizados.
func RemoveTLBEntriesByPID(pid uint) {
	tlbMutex.Lock()
	defer tlbMutex.Unlock()

	filtered := make([]models.TLBEntry, 0, len(tlb))
	for _, entry := range tlb {
		if entry.PID != pid {
			filtered = append(filtered, entry)
		}
	}
	tlb = filtered
}
