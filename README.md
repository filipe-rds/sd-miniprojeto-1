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
├── client_operations.go  # Cliente para testes automatizados e concorrência
├── client_cli.go         # Cliente interativo via terminal
├── go.mod
├── go.sum
├── server.go
├── Dockerfile            # Definição da imagem Docker do servidor
└── docker-compose.yml    # Configuração para rodar o servidor com Docker Compose
```

## Funcionalidades

  * Gerenciamento de múltiplas listas de inteiros identificadas por ID.
  * Operações remotas via RPC: `Append`, `Get`, `Remove`, `Size`. 
  * Persistência do estado por snapshots comprimidos e logs de operações. 
  * Recuperação automática do estado após falhas. 
  * Suporte a acesso concorrente de múltiplos clientes. 
  * Cliente com lógica de reconexão automática e mensagens claras.

## Como Executar o Servidor

Você pode executar o servidor Go diretamente ou utilizando Docker Compose.

### Opção 1: Executar o Servidor Diretamente (Go Native)

1.  **Inicie o servidor:**
    ```sh
    go run server.go
    ```
    Os diretórios `logs/` e `snapshots/` serão criados automaticamente se não existirem. O servidor iniciará e informará a porta em que está escutando.

### Opção 2: Executar o Servidor com Docker Compose

Certifique-se de ter o [Docker Desktop](https://www.docker.com/products/docker-desktop/) (ou Docker Engine e Docker Compose) instalado e rodando em sua máquina.

1.  **Construa a imagem Docker e execute o contêiner do servidor:**
    No diretório raiz do projeto (onde está o `Dockerfile` e o `docker-compose.yml`), execute:

    ```sh
    docker compose up --build -d
    ```

    Os diretórios `logs/` e `snapshots/` serão criados automaticamente no host pelo processo do contêiner se não existirem, devido ao mapeamento de volumes.

      * `--build`: Garante que a imagem seja construída/reconstruída antes de iniciar.
      * `-d`: Roda o contêiner em segundo plano (detached mode).

2.  **Para parar e remover o contêiner (e a rede criada):**

    ```sh
    docker compose down
    ```

3.  **Para remover a imagem Docker gerada (opcional):**
    Após `docker compose down`, a imagem construída pelo Docker Compose pode ou não ser removida automaticamente, dependendo da versão do Docker ou se a imagem for referenciada por outros contêineres. Se você quiser garantir a remoção, use:

    ```sh
    docker rmi remote-list-server
    ```

      * `remote-list-server`: É o nome da imagem definida no `docker-compose.yml` (no bloco `services`, `remote-list-server:`).

## Como Usar o Cliente

Com o servidor em execução (usando Go Native ou Docker Compose), você pode usar os clientes.

### 1\. Use o cliente interativo

Em um **novo terminal**, execute:

```sh
go run client_cli.go
```

**Comandos disponíveis:**

  * `APPEND <list_id> <valor>`: Adiciona um valor ao final da lista. [cite\_start]Ex: `APPEND compras 100` 
  * `GET <list_id> <indice>`: Retorna o valor de um índice específico. [cite\_start]Ex: `GET compras 0` 
  * `REMOVE <list_id>`: Remove e retorna o último valor da lista. [cite\_start]Ex: `REMOVE compras` 
  * `SIZE <list_id>`: Retorna o número de elementos na lista. [cite\_start]Ex: `SIZE compras` 
  * `EXIT`: Sai do cliente.

Este cliente possui lógica de reconexão automática. Tente derrubar e reiniciar o servidor enquanto ele está em uso para observar a reconexão.

### 2\. Teste operações automáticas e concorrência

Em um **novo terminal**, execute:

```sh
go run client_operations.go
```

Este script executará uma série de operações pré-definidas, incluindo um teste de concorrência que simula múltiplos clientes acessando as listas simultaneamente. Observe os logs do servidor para ver a interação.

## Estruturas Principais

* `RemoteList`: Gerencia todas as listas ativas no servidor.

* `SpecificList`: Representa uma única lista de inteiros.

* Estruturas de argumentos para RPC: `AppendArgs`, `GetArgs`, `RemoveArgs`, `SizeArgs`.


## Persistência

* Snapshots comprimidos (`.gz`) em `snapshots/remote_list_snapshot.json.gz`.

* Logs de operações (Write-Ahead Log) em `logs/operations.log`.

* Utilitários em `utils/processing_snapshots.go` (salvar/carregar snapshots) e `utils/processing_logs.go` (gravar/ler logs). 