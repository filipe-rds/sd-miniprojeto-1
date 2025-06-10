package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"

	"sd-miniprojeto-1/structures"
	"sd-miniprojeto-1/utils"
)

const (
	snapshotInterval = 10 * time.Second // Intervalo entre salvamentos de snapshots.
	serverPort       = ":1234"          // Porta do servidor RPC.
)

// RemoteListService atende aos pedidos dos clientes via RPC.
type RemoteListService struct {
	remoteList                   *structures.RemoteList // Gerencia os dados das listas.
	lastLoggedOperationTimestamp time.Time              // Horário da última operação salva no log.
	logTimestampMutex            sync.Mutex             // Protege o acesso ao timestamp do log.
}

// logAndTrackWithValue registra operações com valor/índice e atualiza o timestamp do log.
func (s *RemoteListService) logAndTrackWithValue(logFunc func(string, int) error, listID string, val int) error {
	operationTime := time.Now()
	err := logFunc(listID, val)
	if err != nil {
		return err
	}

	s.logTimestampMutex.Lock()
	if operationTime.After(s.lastLoggedOperationTimestamp) {
		s.lastLoggedOperationTimestamp = operationTime
	}
	s.logTimestampMutex.Unlock()
	return nil
}

// logAndTrackWithoutValue registra operações sem valor/índice e atualiza o timestamp do log.
func (s *RemoteListService) logAndTrackWithoutValue(logFunc func(string) error, listID string) error {
	operationTime := time.Now()
	err := logFunc(listID)
	if err != nil {
		return err
	}

	s.logTimestampMutex.Lock()
	if operationTime.After(s.lastLoggedOperationTimestamp) {
		s.lastLoggedOperationTimestamp = operationTime
	}
	s.logTimestampMutex.Unlock()
	return nil
}

// Append é o método RPC para adicionar um valor a uma lista.
func (s *RemoteListService) Append(args structures.AppendArgs, reply *bool) error {
	if err := s.logAndTrackWithValue(utils.AppendLog, args.ListID, args.Value); err != nil {
		log.Printf("Erro ao logar APPEND para ListaID %s, Valor %d: %v", args.ListID, args.Value, err)
	}
	return s.remoteList.Append(args, reply)
}

// Get é o método RPC para obter um valor de uma lista.
func (s *RemoteListService) Get(args structures.GetArgs, reply *int) error {
	if err := s.logAndTrackWithValue(utils.GetLog, args.ListID, args.Index); err != nil {
		log.Printf("Erro ao logar GET para ListaID %s, Índice %d: %v", args.ListID, args.Index, err)
	}
	return s.remoteList.Get(args, reply)
}

// Remove é o método RPC para remover o último elemento de uma lista.
func (s *RemoteListService) Remove(args structures.RemoveArgs, reply *int) error {
	if err := s.logAndTrackWithoutValue(utils.RemoveLog, args.ListID); err != nil {
		log.Printf("Erro ao logar REMOVE para ListaID %s: %v", args.ListID, err)
	}
	return s.remoteList.Remove(args, reply)
}

// Size é o método RPC para obter o tamanho de uma lista.
func (s *RemoteListService) Size(args structures.SizeArgs, reply *int) error {
	if err := s.logAndTrackWithValue(utils.GetLog, args.ListID, 0); err != nil {
		log.Printf("Erro ao logar SIZE para ListaID %s: %v", args.ListID, err)
	}
	return s.remoteList.Size(args, reply)
}

func main() {
	// 0. Prepara as pastas para logs e snapshots.
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("Falha ao criar diretório 'logs': %v", err)
	}
	if err := os.MkdirAll("snapshots", 0755); err != nil {
		log.Fatalf("Falha ao criar diretório 'snapshots': %v", err)
	}

	// 1. Carrega dados de snapshot e logs para recuperar o estado.
	fmt.Println("Tentando carregar snapshot...")
	remoteList, lastSnapshotLogTimestamp, err := utils.LoadSnapshot()
	if err != nil {
		log.Fatalf("Erro ao carregar snapshot: %v", err)
	}
	fmt.Println("Snapshot carregado ou nova lista criada.")

	remoteListService := &RemoteListService{
		remoteList:                   remoteList,
		lastLoggedOperationTimestamp: lastSnapshotLogTimestamp,
	}

	// Lógica de Recuperação de Dados.
	recoveryStartTime := remoteListService.lastLoggedOperationTimestamp
	if recoveryStartTime.IsZero() {
		fmt.Println("Nenhum timestamp de snapshot válido. Aplicando todos os logs disponíveis.")
	} else {
		fmt.Printf("Aplicando logs a partir de %s...\n", recoveryStartTime.Format(time.RFC3339Nano))
	}

	logEntries, err := utils.ReadLogsFromTimestamp(recoveryStartTime)
	if err != nil {
		log.Printf("Erro ao ler logs para recuperação: %v.", err)
	} else {
		for _, entry := range logEntries {
			// Reaplica operações do log diretamente na lista (sem logar novamente).
			switch entry.Operation {
			case "Append":
				remoteList.Append(structures.AppendArgs{ListID: entry.ListID, Value: entry.Value}, new(bool))
			case "Remove":
				remoteList.Remove(structures.RemoveArgs{ListID: entry.ListID}, new(int))
			}
			// Atualiza o timestamp da última operação reaplicada.
			remoteListService.logTimestampMutex.Lock()
			if entry.Timestamp.After(remoteListService.lastLoggedOperationTimestamp) {
				remoteListService.lastLoggedOperationTimestamp = entry.Timestamp
			}
			remoteListService.logTimestampMutex.Unlock()
		}
		fmt.Printf("%d logs relevantes aplicados.\n", len(logEntries))
	}

	// 3. Prepara e registra o serviço RPC.
	err = rpc.RegisterName("RemoteList", remoteListService)
	if err != nil {
		log.Fatalf("Falha ao registrar serviço RPC: %v", err)
	}
	rpc.HandleHTTP() // Configura o RPC sobre HTTP.

	// 4. Começa a escutar por conexões.
	listener, err := net.Listen("tcp", serverPort)
	if err != nil {
		log.Fatalf("Falha ao escutar na porta %s: %v", serverPort, err)
	}
	fmt.Printf("Servidor online na porta %s...\n", serverPort)

	// 5. Inicia salvamento periódico de snapshots em segundo plano.
	ticker := time.NewTicker(snapshotInterval)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			fmt.Println("Tentando salvar snapshot...")
			remoteListService.logTimestampMutex.Lock()
			currentLastLogTimestamp := remoteListService.lastLoggedOperationTimestamp
			remoteListService.logTimestampMutex.Unlock()

			err := utils.SaveSnapshot(remoteList, currentLastLogTimestamp)
			if err != nil {
				log.Printf("Erro ao salvar snapshot: %v", err)
			}
		}
	}()

	// 6. Servidor atende às requisições.
	log.Fatal(http.Serve(listener, nil))
}