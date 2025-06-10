package utils

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LogEntry representa uma entrada no log de operações.
type LogEntry struct {
	Timestamp time.Time // Horário da operação.
	Operation string    // Tipo de operação (e.g., "Append").
	ListID    string    // ID da lista.
	Value     int       // Valor envolvido (para Append).
	Index     int       // Índice envolvido (para Get).
}

const (
	logsDir     = "logs"          // Diretório de logs.
	logFileName = "operations.log" // Nome do arquivo de log.
)

// getLogFilePath retorna o caminho completo do arquivo de log.
func getLogFilePath() string {
	return filepath.Join(logsDir, logFileName)
}

// AppendLog escreve uma entrada de log para operações de adição.
func AppendLog(listID string, value int) error {
	return writeLog(LogEntry{
		Timestamp: time.Now(),
		Operation: "Append",
		ListID:    listID,
		Value:     value,
	})
}

// RemoveLog escreve uma entrada de log para operações de remoção.
func RemoveLog(listID string) error {
	return writeLog(LogEntry{
		Timestamp: time.Now(),
		Operation: "Remove",
		ListID:    listID,
	})
}

// GetLog escreve uma entrada de log para operações de leitura.
func GetLog(listID string, valueOrIndex int) error {
	return writeLog(LogEntry{
		Timestamp: time.Now(),
		Operation: "Get/Size",
		ListID:    listID,
		Index:     valueOrIndex,
	})
}

// writeLog escreve uma entrada no arquivo de log.
func writeLog(entry LogEntry) error {
	fileName := getLogFilePath()
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("erro ao abrir arquivo de log %s: %w", fileName, err)
	}
	defer f.Close()

	fileLogger := log.New(f, "", 0)

	logString := fmt.Sprintf("%s %s %s", entry.Timestamp.Format(time.RFC3339Nano), entry.Operation, entry.ListID)
	switch entry.Operation {
	case "Append":
		logString += fmt.Sprintf(" %d", entry.Value)
	case "Get/Size":
		logString += fmt.Sprintf(" %d", entry.Index)
	}

	fileLogger.Println(logString)
	return nil
}

// ReadLogsFromTimestamp lê entradas de log a partir de um timestamp específico.
func ReadLogsFromTimestamp(since time.Time) ([]LogEntry, error) {
	fileName := getLogFilePath()
	f, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, fmt.Errorf("erro ao abrir arquivo de log: %w", err)
	}
	defer f.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 3 {
			log.Printf("Pulando linha de log malformada %d: %s", lineNum, line)
			continue
		}

		timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			log.Printf("Pulando linha de log %d (parse timestamp): %v", lineNum, err)
			continue
		}

		if timestamp.After(since) { // Inclui apenas logs APÓS o timestamp fornecido.
			entry := LogEntry{Timestamp: timestamp, Operation: parts[1], ListID: parts[2]}
			switch entry.Operation {
			case "Append":
				if len(parts) > 3 {
					val, err := strconv.Atoi(parts[3])
					if err != nil {
						log.Printf("Pulando linha de log %d (parse valor): %v", lineNum, err)
						continue
					}
					entry.Value = val
				} else {
					log.Printf("Pulando linha de log Append malformada %d: valor ausente", lineNum)
					continue
				}
			case "Get/Size":
				if len(parts) > 3 {
					idx, err := strconv.Atoi(parts[3])
					if err != nil {
						log.Printf("Pulando linha de log %d (parse índice): %v", lineNum, err)
						continue
					}
					entry.Index = idx
				}
			}
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, fmt.Errorf("erro ao ler arquivo de log: %w", err)
	}

	return entries, nil
}