package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"sync"
	"time"

	"sd-miniprojeto-1/structures" // Importa as definições das estruturas de dados.
)

const serverAddress = "localhost:1234" // Endereço do servidor RPC.

func main() {
	// Conecta ao servidor RPC.
	client, err := rpc.DialHTTP("tcp", serverAddress)
	if err != nil {
		log.Fatal("Erro ao conectar ao servidor:", err)
	}
	defer client.Close() // Garante que a conexão seja fechada ao sair.

	listID1 := "minha_lista_1"
	listID2 := "outra_lista"

	// --- Testes de Operações Básicas (Pré-concorrência) ---
	fmt.Printf("\n--- Teste: Operações Básicas ---\n")
	var replyBool bool

	// Teste: Append
	err = client.Call("RemoteList.Append", structures.AppendArgs{ListID: listID1, Value: 10}, &replyBool)
	if err != nil {
		log.Fatal("Erro no Append:", err)
	}
	fmt.Printf("Append %d em %s: %t\n", 10, listID1, replyBool)

	err = client.Call("RemoteList.Append", structures.AppendArgs{ListID: listID1, Value: 20}, &replyBool)
	if err != nil {
		log.Fatal("Erro no Append:", err)
	}
	fmt.Printf("Append %d em %s: %t\n", 20, listID1, replyBool)

	err = client.Call("RemoteList.Append", structures.AppendArgs{ListID: listID2, Value: 5}, &replyBool)
	if err != nil {
		log.Fatal("Erro no Append:", err)
	}
	fmt.Printf("Append %d em %s: %t\n", 5, listID2, replyBool)

	// Teste: Size
	var size int
	err = client.Call("RemoteList.Size", structures.SizeArgs{ListID: listID1}, &size)
	if err != nil {
		log.Fatal("Erro no Size:", err)
	}
	fmt.Printf("Tamanho de %s: %d\n", listID1, size)

	err = client.Call("RemoteList.Size", structures.SizeArgs{ListID: listID2}, &size)
	if err != nil {
		log.Fatal("Erro no Size:", err)
	}
	fmt.Printf("Tamanho de %s: %d\n", listID2, size)

	// Teste: Get
	var value int
	err = client.Call("RemoteList.Get", structures.GetArgs{ListID: listID1, Index: 0}, &value)
	if err != nil {
		log.Fatal("Erro no Get:", err)
	}
	fmt.Printf("Get de %s no índice 0: %d\n", listID1, value)

	err = client.Call("RemoteList.Get", structures.GetArgs{ListID: listID1, Index: 1}, &value)
	if err != nil {
		log.Fatal("Erro no Get:", err)
	}
	fmt.Printf("Get de %s no índice 1: %d\n", listID1, value)

	// Teste: Remove
	var removedValue int
	err = client.Call("RemoteList.Remove", structures.RemoveArgs{ListID: listID1}, &removedValue)
	if err != nil {
		log.Fatal("Erro no Remove:", err)
	}
	fmt.Printf("Removido de %s: %d\n", listID1, removedValue)

	err = client.Call("RemoteList.Size", structures.SizeArgs{ListID: listID1}, &size)
	if err != nil {
		log.Fatal("Erro no Size após Remove:", err)
	}
	fmt.Printf("Tamanho de %s após Remove: %d\n", listID1, size)

	// Teste: Acessar lista não existente ou índice inválido.
	fmt.Printf("\n--- Teste: Erros Esperados ---\n")
	err = client.Call("RemoteList.Size", structures.SizeArgs{ListID: "nao_existe"}, &size)
	if err != nil {
		fmt.Printf("Erro esperado para lista inexistente: %v\n", err)
	} else {
		fmt.Println("Sucesso inesperado para lista inexistente.")
	}

	// --- Seção de Concorrência ---
	fmt.Printf("\n--- Teste: Concorrência Simplificada ---\n")
	numConcurrentClients := 3  // Número de clientes (goroutines) concorrentes.
	operationsPerClient := 10  // Operações por cliente.
	
	var wg sync.WaitGroup // Usado para esperar todas as goroutines terminarem.
	concurrentListID := "lista_concorrente_simples"

	// Garante que a lista concorrente exista com um valor inicial.
	_ = client.Call("RemoteList.Append", structures.AppendArgs{ListID: concurrentListID, Value: 0}, &replyBool)
	
	fmt.Printf("Iniciando %d clientes concorrentes (%d operações/cliente)...\n", numConcurrentClients, operationsPerClient)
	
	rand.Seed(time.Now().UnixNano()) // Inicializa gerador de números aleatórios.

	for i := 0; i < numConcurrentClients; i++ {
		wg.Add(1) // Adiciona 1 ao contador de goroutines.
		go func(clientID int) {
			defer wg.Done() // Garante que o contador seja decrementado ao final da goroutine.
			
			// Cada goroutine cria sua própria conexão RPC.
			localClient, localErr := rpc.DialHTTP("tcp", serverAddress)
			if localErr != nil {
				log.Printf("Cliente %d: Falha ao conectar ao servidor: %v", clientID, localErr)
				return
			}
			defer localClient.Close()

			for j := 0; j < operationsPerClient; j++ {
				opType := rand.Intn(100) // 0-99 para decidir a operação.

				if opType < 50 { // 50% Append
					valueToAppend := (clientID * 1000) + j // Valores únicos por cliente.
					var rb bool
					_ = localClient.Call("RemoteList.Append", structures.AppendArgs{ListID: concurrentListID, Value: valueToAppend}, &rb)
				} else if opType < 75 { // 25% Get
					var val int
					currentSize := 0
					_ = localClient.Call("RemoteList.Size", structures.SizeArgs{ListID: concurrentListID}, &currentSize)
					if currentSize > 0 {
						randomIndex := rand.Intn(currentSize)
						_ = localClient.Call("RemoteList.Get", structures.GetArgs{ListID: concurrentListID, Index: randomIndex}, &val)
					}
				} else { // 25% Remove
					var removedVal int
					_ = localClient.Call("RemoteList.Remove", structures.RemoveArgs{ListID: concurrentListID}, &removedVal)
				}
				// Pequeno atraso aleatório para variar a concorrência.
				time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
			}
		}(i) // Passa o ID do cliente para a goroutine.
	}

	wg.Wait() // Espera todas as goroutines de clientes terminarem.

	// --- Verificações Pós-Concorrência ---
	fmt.Println("\n--- Verificações Pós-Concorrência ---")

	var finalSize int
	err = client.Call("RemoteList.Size", structures.SizeArgs{ListID: concurrentListID}, &finalSize)
	if err != nil {
		log.Fatalf("Falha ao obter tamanho final de %s: %v", concurrentListID, err)
	}
	fmt.Printf("Tamanho final da lista '%s' após operações concorrentes: %d\n", concurrentListID, finalSize)
	
	if finalSize < 0 {
		log.Fatalf("ERRO CRÍTICO: Tamanho da lista negativo! Indicação de corrupção.")
	}

	if finalSize > 0 {
		var firstElement int
		err = client.Call("RemoteList.Get", structures.GetArgs{ListID: concurrentListID, Index: 0}, &firstElement)
		if err != nil {
			log.Fatalf("ERRO: Não foi possível obter o primeiro elemento da lista concorrente: %v", err)
		}
		fmt.Printf("Primeiro elemento da lista '%s': %d\n", concurrentListID, firstElement)
	} else {
		fmt.Printf("A lista '%s' está vazia após operações concorrentes.\n", concurrentListID)
	}
	
	fmt.Println("Teste de concorrência concluído.")
	fmt.Println("\n--- Teste Geral Concluído ---")
}