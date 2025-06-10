
# Mini Projeto 1 - Sistemas Distribuídos

Projeto de sistema distribuído simples para gerenciamento remoto de listas de inteiros via RPC, com persistência por snapshots e logs.

## Estrutura de Pastas

```markdown
├── logs/
│   └── operations.log
├── snapshots/
│   └── remote_list_snapshot.json.gz
├── structures/
│   └── remote_list.go
├── utils/
│   ├── processing_logs.go
│   └── processing_snapshots.go
├── client_operations.go
├── client.go
├── go.mod
└── server.go
```

## Funcionalidades

- Gerenciamento de múltiplas listas de inteiros identificadas por ID.
- Operações remotas via RPC: Append, Get, Remove, Size.
- Persistência do estado por snapshots comprimidos e logs de operações.
- Recuperação automática do estado após falhas.
- Suporte a concorrência e múltiplos clientes.

## Como Executar

### 1. Inicie o servidor

```sh
go run server.go
```

### 2. Use o cliente interativo

Em outro terminal:

```sh
go run client.go
```

Comandos disponíveis:
- `APPEND <list_id> <valor>`
- `GET <list_id> <indice>`
- `REMOVE <list_id>`
- `SIZE <list_id>`
- `EXIT`

### 3. Teste operações automáticas e concorrência

```sh
go run client_operations.go
```

## Estruturas Principais

- `RemoteList`: Gerencia todas as listas.
- `SpecificList`: Representa uma lista individual.
- Estruturas de argumentos para RPC: `AppendArgs`, `GetArgs`, `RemoveArgs`, `SizeArgs`.

## Persistência

- Snapshots comprimidos em `snapshots/remote_list_snapshot.json.gz`.
- Logs de operações em `logs/operations.log`.
- Utilitários em `utils/processing_snapshots.go` e `utils/processing_logs.go`.

---