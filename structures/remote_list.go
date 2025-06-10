package structures

import (
	"fmt"
	"sync"
)

// RemoteList gerencia coleções de listas de inteiros por um ID.
type RemoteList struct {
	Lists map[string]*SpecificList // Mapa de IDs de listas para listas específicas.
	Mu    sync.RWMutex             `json:"-"` // Mutex para o mapa 'Lists'. Ignorado no JSON.
}

// SpecificList representa uma única lista de valores inteiros.
type SpecificList struct {
	Elements []int      // Elementos da lista.
	mu       sync.Mutex // Mutex para a lista específica.
}

// NewRemoteList cria uma nova instância de RemoteList.
func NewRemoteList() *RemoteList {
	return &RemoteList{
		Lists: make(map[string]*SpecificList),
		Mu:    sync.RWMutex{},
	}
}

// NewSpecificList cria uma nova lista com mutex inicializado.
func NewSpecificList(elements []int) *SpecificList {
    return &SpecificList{
        Elements: elements,
        mu:       sync.Mutex{},
    }
}

// ensureListExists garante que uma lista exista no dicionário.
// Retorna a lista (existente ou nova).
func (rl *RemoteList) ensureListExists(listID string) *SpecificList {
	if _, ok := rl.Lists[listID]; !ok {
		rl.Lists[listID] = NewSpecificList(make([]int, 0))
	}
	return rl.Lists[listID]
}

// --- Tipos de Argumentos RPC ---

// AppendArgs para o método Append.
type AppendArgs struct {
	ListID string
	Value  int
}

// GetArgs para o método Get.
type GetArgs struct {
	ListID string
	Index  int
}

// RemoveArgs para o método Remove.
type RemoveArgs struct {
	ListID string
}

// SizeArgs para o método Size.
type SizeArgs struct {
	ListID string
}

// --- Métodos RPC ---

// Append adiciona um valor ao final da lista.
func (rl *RemoteList) Append(args AppendArgs, reply *bool) error {
	rl.Mu.Lock()
	specificList := rl.ensureListExists(args.ListID)
	rl.Mu.Unlock()

	specificList.mu.Lock()
	specificList.Elements = append(specificList.Elements, args.Value)
	specificList.mu.Unlock()

	*reply = true
	return nil
}

// Get retorna um valor em uma posição específica da lista.
func (rl *RemoteList) Get(args GetArgs, reply *int) error {
	rl.Mu.RLock()
	specificList, ok := rl.Lists[args.ListID]
	rl.Mu.RUnlock()

	if !ok {
		return fmt.Errorf("lista com ID '%s' não encontrada", args.ListID)
	}

	specificList.mu.Lock()
	defer specificList.mu.Unlock()

	if args.Index < 0 || args.Index >= len(specificList.Elements) {
		return fmt.Errorf("índice %d fora dos limites para a lista ID '%s' (tamanho %d)", args.Index, args.ListID, len(specificList.Elements))
	}

	*reply = specificList.Elements[args.Index]
	return nil
}

// Remove remove e retorna o último elemento da lista.
func (rl *RemoteList) Remove(args RemoveArgs, reply *int) error {
	rl.Mu.RLock()
	specificList, ok := rl.Lists[args.ListID]
	rl.Mu.RUnlock()

	if !ok {
		return fmt.Errorf("lista com ID '%s' não encontrada", args.ListID)
	}

	specificList.mu.Lock()
	defer specificList.mu.Unlock()

	if len(specificList.Elements) == 0 {
		return fmt.Errorf("lista com ID '%s' está vazia", args.ListID)
	}

	lastIndex := len(specificList.Elements) - 1
	*reply = specificList.Elements[lastIndex]
	specificList.Elements = specificList.Elements[:lastIndex]
	return nil
}

// Size obtém a quantidade de elementos na lista.
func (rl *RemoteList) Size(args SizeArgs, reply *int) error {
	rl.Mu.RLock()
	specificList, ok := rl.Lists[args.ListID]
	rl.Mu.RUnlock()

	if !ok {
		return fmt.Errorf("lista com ID '%s' não encontrada", args.ListID)
	}

	specificList.mu.Lock()
	defer specificList.mu.Unlock()

	*reply = len(specificList.Elements)
	return nil
}