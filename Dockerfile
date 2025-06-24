# Fase de Build: Compila a aplicação Go.
FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./server.go

# Fase Final: Cria a imagem Docker leve com o binário compilado.
FROM scratch
WORKDIR /app
COPY --from=builder /app/server .
CMD ["./server"]