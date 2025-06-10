package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"

	"sd-miniprojeto-1/structures"
)

const (
	serverAddress       = "localhost:1234"
	reconnectionTimeout = 30 * time.Second // Tempo máximo para tentar reconectar.
	retryDelay          = 2 * time.Second  // Atraso entre tentativas de reconexão.
)

// clientConn guarda a conexão ativa com o servidor.
var clientConn *rpc.Client

// tryConnect tenta conectar ao servidor uma vez.
func tryConnect() (*rpc.Client, error) {
	c, err := rpc.DialHTTP("tcp", serverAddress)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// isConnectionError verifica se um erro indica falha na conexão.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, io.EOF) ||
		strings.Contains(err.Error(), "connection reset by peer") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "connection is shut down") ||
		strings.Contains(err.Error(), "i/o timeout") {
		return true
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	
	// Erros RPC que não são de conexão (ex: método não encontrado).
	if strings.HasPrefix(err.Error(), "rpc:") {
		return false
	}

	return false
}

// ensureConnected garante uma conexão ativa, reconectando se necessário.
func ensureConnected(initialCall bool) bool {
	if clientConn != nil {
		var dummyValue int
		err := clientConn.Call("RemoteList.Get", structures.GetArgs{ListID: "lista_dummy_ping"}, &dummyValue)
		
		if err == nil || !isConnectionError(err) {
			return true // Conexão ativa ou erro de lógica.
		}
		
		fmt.Println("Conexão RPC inativa. Tentando reconectar...")
		clientConn.Close()
		clientConn = nil
	}

	startTime := time.Now()
	attemptCount := 0

	for time.Since(startTime) < reconnectionTimeout {
		attemptCount++

		if !initialCall || (initialCall && attemptCount > 1) {
			remainingTime := reconnectionTimeout - time.Since(startTime).Truncate(time.Second)
			fmt.Printf("Tentando conectar... Tempo restante: %v\n", remainingTime)
		}

		c, err := tryConnect()
		if err == nil {
			fmt.Println("Conexão estabelecida!")
			clientConn = c
			return true
		}
		
		if time.Since(startTime) + retryDelay < reconnectionTimeout {
			fmt.Printf("Erro na conexão: %v. Tentando novamente em %v...\n", err, retryDelay)
			time.Sleep(retryDelay)
		} else {
			fmt.Printf("Erro na conexão: %v. Tempo limite de reconexão atingido.\n", err)
		}
	}

	fmt.Printf("Não foi possível conectar/reconectar após %v. Verifique se o servidor está em execução.\n", reconnectionTimeout)
	return false
}

// callRPC faz uma chamada RPC, cuidando de reconexão e erros.
func callRPC(serviceMethod string, args interface{}, reply interface{}) error {
	if !ensureConnected(false) {
		return fmt.Errorf("não foi possível estabelecer conexão RPC")
	}

	err := clientConn.Call(serviceMethod, args, reply)
	if err != nil {
		if isConnectionError(err) {
			fmt.Printf("Erro na chamada RPC (%s), conexão perdida: %v. Tentando reconectar e refazer...\n", serviceMethod, err)
			
			if clientConn != nil {
				clientConn.Close()
				clientConn = nil
			}
			
			if !ensureConnected(false) {
				return fmt.Errorf("falha ao refazer chamada RPC após reconexão: %v", err)
			}
			
			err = clientConn.Call(serviceMethod, args, reply)
			if err != nil {
				return fmt.Errorf("falha na segunda tentativa de chamada RPC: %v", err)
			}
		} else {
			return fmt.Errorf("erro RPC de negócio: %v", err)
		}
	}
	return nil
}

func main() {
	fmt.Println("Bem-vindo ao Cliente RemoteList RPC!")
	fmt.Println("Comandos disponíveis:")
	fmt.Println("  APPEND <list_id> <valor>")
	fmt.Println("  GET <list_id> <indice>")
	fmt.Println("  REMOVE <list_id>")
	fmt.Println("  SIZE <list_id>")
	fmt.Println("  EXIT (para sair)")
	fmt.Println("---------------------------------")

	reader := bufio.NewReader(os.Stdin)

	if !ensureConnected(true) { 
		os.Exit(1)
	}

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		parts := strings.Fields(input)

		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])

		var err error 

		switch command {
		case "APPEND":
			if len(parts) != 3 {
				fmt.Println("Uso: APPEND <list_id> <valor>")
				continue
			}
			listID := parts[1]
			value, parseErr := strconv.Atoi(parts[2])
			if parseErr != nil {
				fmt.Println("Erro: Valor deve ser um número inteiro.", parseErr)
				continue
			}

			var replyBool bool
			err = callRPC("RemoteList.Append", structures.AppendArgs{ListID: listID, Value: value}, &replyBool)
			if err != nil {
				fmt.Printf("Erro no APPEND: %v\n", err)
			} else {
				fmt.Printf("Sucesso: Valor %d adicionado à lista %s\n", value, listID)
			}

		case "GET":
			if len(parts) != 3 {
				fmt.Println("Uso: GET <list_id> <indice>")
				continue
			}
			listID := parts[1]
			index, parseErr := strconv.Atoi(parts[2])
			if parseErr != nil {
				fmt.Println("Erro: Índice deve ser um número inteiro.", parseErr)
				continue
			}

			var value int
			err = callRPC("RemoteList.Get", structures.GetArgs{ListID: listID, Index: index}, &value)
			if err != nil {
				fmt.Printf("Erro no GET: %v\n", err)
			} else {
				fmt.Printf("Sucesso: Lista %s, Índice %d -> Valor: %d\n", listID, index, value)
			}

		case "REMOVE":
			if len(parts) != 2 {
				fmt.Println("Uso: REMOVE <list_id>")
				continue
			}
			listID := parts[1]

			var removedValue int
			err = callRPC("RemoteList.Remove", structures.RemoveArgs{ListID: listID}, &removedValue)
			if err != nil {
				fmt.Printf("Erro no REMOVE: %v\n", err)
			} else {
				fmt.Printf("Sucesso: Valor %d removido da lista %s\n", removedValue, listID)
			}

		case "SIZE":
			if len(parts) != 2 {
				fmt.Println("Uso: SIZE <list_id>")
				continue
			}
			listID := parts[1]

			var size int
			err = callRPC("RemoteList.Size", structures.SizeArgs{ListID: listID}, &size)
			if err != nil {
				fmt.Printf("Erro no SIZE: %v\n", err)
			} else {
				fmt.Printf("Sucesso: Lista %s -> Tamanho: %d\n", listID, size)
			}

		case "EXIT":
			fmt.Println("Saindo do cliente.")
			if clientConn != nil {
				clientConn.Close()
			}
			return

		default:
			fmt.Println("Comando desconhecido. Use APPEND, GET, REMOVE, SIZE ou EXIT.")
		}
	}
}