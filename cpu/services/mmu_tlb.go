package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
	"net/http"
)

var (
	tlb          []models.TLBEntry
	tlbMaxSize   int
	tlbAlgorithm string // "FIFO" o "LRU"
	tlbCounter   int64  // para LRU, contador incremental
	tlbMutex     sync.Mutex
)

func InitTLB() {
	tlbMaxSize = models.CpuConfig.TlbEntries
	tlbAlgorithm = models.CpuConfig.TlbReplacement // "FIFO" o "LRU"
	tlbCounter = 0
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

func TranslateAddress(pid int, logicalAddress int) int {
	pageSize := models.MemConfig.PageSize
	pageNumber := logicalAddress / pageSize
	offset := logicalAddress % pageSize

	//Verifica que la tlb no este desactivada
	if tlbMaxSize > 0 {
		if frame, ok := searchTLB(pid, pageNumber); ok {
			//si la encuentra imprime TLB HIT y traduce
			slog.Info(fmt.Sprintf("PID: %d - TLB HIT - Pagina: %d", pid, pageNumber))
			return frame*pageSize + offset
		}
		slog.Info(fmt.Sprintf("PID: %d - TLB MISS - Página: %d", pid, pageNumber))
		frame := tlb_miss(pid, pageNumber)
		insert_tlb(pid, pageNumber, frame)
		return frame*pageSize + offset
	}
	// TLB desactivada
	slog.Info(fmt.Sprintf("PID: %d - TLB desactivada - Traducción completa - Pagina: %d", pid, pageNumber))
	frame := tlb_miss(pid, pageNumber)
	return frame*pageSize + offset

}

func tlb_miss(pid int, pageNumber int) int {
	numLevels := models.MemConfig.NumberOfLevels
	entriesPerPage := models.MemConfig.EntriesPerPage

	var entries []int
	for level := 1; level <= numLevels; level++ {
		entry := (pageNumber / intPow(entriesPerPage, numLevels-level)) % entriesPerPage
		entries = append(entries, entry)
	}
	return RequestMemoryFrame(pid, entries)
}

func searchTLB(pid int, pagina int) (int, bool) {
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

func insert_tlb(pid int, pagina int, frame int) {
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
	if tlbAlgorithm == "FIFO" {
		victimIndex = 0
	} else if tlbAlgorithm == "LRU" {
		minUsage := tlb[0].LastUsed
		victimIndex = 0
		for i, e := range tlb {
			if e.LastUsed < minUsage {
				minUsage = e.LastUsed
				victimIndex = i
			}
		}
	}

	tlb[victimIndex] = entry
}

func RequestMemoryFrame(pid int, entries []int) int {
	type Request struct {
		PID     int   `json:"pid"`
		Entries []int `json:"entries"`
	}
	type Response struct {
		Frame int `json:"frame"`
	}

	reqBody := Request{PID: pid, Entries: entries}
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

	return decoded.Frame
}

func intPow(base, exp int) int {
	result := 1
	for exp > 0 {
		result *= base
		exp--
	}
	return result
}

func WriteToMemory(pid int, address int, data string, config *models.Config) error {
	writeReq := models.WriteRequest{
		PID:             pid,
		LogicalAddr:     address,
		PhysicalAddress: TranslateAddress(pid, address),
		Data:            data,
	}

	body, err := json.Marshal(writeReq)
	if err != nil {
		slog.Error("Failed to serialize write request", "error", err)
		return err
	}
	slog.Debug("Write request sent to memory",
		"address", address,
		"data_length", len(data),
	)

	//PETICION HTTP
	resp, err := client.DoRequest(
		config.PortMemory,
		config.IpMemory,
		"POST",
		"memoria/write",
		body,
	)
	if err != nil || resp.StatusCode != http.StatusOK {
		slog.Error("Memory write failed",
			"error", err,
			"status_code", resp.StatusCode,
		)
		return err
	}

	return nil
}
