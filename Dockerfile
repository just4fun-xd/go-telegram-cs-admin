# Этап сборки – используем образ на базе Debian Bullseye
FROM golang:1.22-bullseye AS builder
WORKDIR /app

# Устанавливаем зависимости для CGO и SQLite
RUN apt-get update && \
    apt-get install -y gcc libsqlite3-dev && \
    rm -rf /var/lib/apt/lists/*

# Копируем файлы зависимостей и скачиваем модули
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник для архитектуры AMD64 с включенным CGO
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bot ./cmd/bot

# Этап исполнения – минимальный образ Debian
FROM debian:bullseye-slim
WORKDIR /root/

# Устанавливаем runtime-зависимости для SQLite3
RUN apt-get update && \
    apt-get install -y libsqlite3-0 ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Копируем собранное приложение из этапа сборки
COPY --from=builder /app/bot .

# Запускаем приложение
ENTRYPOINT ["./bot"]
