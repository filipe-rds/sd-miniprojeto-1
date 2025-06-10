package utils

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sd-miniprojeto-1/structures"
)

const (
	snapshotsDir     = "snapshots"          // Diretório de snapshots.
	snapshotFileName = "remote_list_snapshot.json.gz" // Nome do arquivo de snapshot.
)

// SnapshotContent armazena o estado do RemoteList e o timestamp do último log coberto.
type SnapshotContent struct {
	RemoteList       *structures.RemoteList // Estado das listas.
	LastLogTimestamp time.Time            // Timestamp do último log coberto.
}

// SaveSnapshot salva o RemoteList e o timestamp do último log em um arquivo comprimido.
func SaveSnapshot(rl *structures.RemoteList, lastLogTimestamp time.Time) error {
	filePath := filepath.Join(snapshotsDir, snapshotFileName)

	rl.Mu.RLock() // Protege o RemoteList para leitura consistente.
	defer rl.Mu.RUnlock()

	content := SnapshotContent{RemoteList: rl, LastLogTimestamp: lastLogTimestamp}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo de snapshot %s: %w", filePath, err)
	}
	defer f.Close()

	writer := gzip.NewWriter(f)
	defer writer.Close()

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(content); err != nil {
		return fmt.Errorf("erro ao codificar conteúdo do snapshot: %w", err)
	}

	fmt.Printf("Snapshot salvo e comprimido em %s (Cobre logs até: %s).\n", filePath, lastLogTimestamp.Format(time.RFC3339))
	return nil
}

// LoadSnapshot carrega o RemoteList e o timestamp do último log do arquivo comprimido.
func LoadSnapshot() (*structures.RemoteList, time.Time, error) {
	filePath := filepath.Join(snapshotsDir, snapshotFileName)

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Nenhum snapshot encontrado. Iniciando com um RemoteList vazio.")
			return structures.NewRemoteList(), time.Time{}, nil
		}
		return nil, time.Time{}, fmt.Errorf("erro ao abrir arquivo de snapshot %s: %w", filePath, err)
	}
	defer f.Close()

	reader, err := gzip.NewReader(f)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("erro ao criar leitor gzip para snapshot %s: %w", filePath, err)
	}
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	var content SnapshotContent
	if err := decoder.Decode(&content); err != nil {
		return nil, time.Time{}, fmt.Errorf("erro ao decodificar conteúdo do snapshot de %s: %w", filePath, err)
	}

	// Reinicializa mutexes das listas carregadas.
	reconstructedLists := make(map[string]*structures.SpecificList)
	for listID, loadedList := range content.RemoteList.Lists {
		reconstructedLists[listID] = structures.NewSpecificList(loadedList.Elements)
	}
	content.RemoteList.Lists = reconstructedLists

	fmt.Printf("Snapshot carregado de %s (Cobre logs até: %s).\n", filePath, content.LastLogTimestamp.Format(time.RFC3339))
	return content.RemoteList, content.LastLogTimestamp, nil
}