# Сборка приложения
FROM golang:1.27.4-alpine3.24 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /screener ./cmd/screener

# Сборка Goose с поддержкой PostgreSQL
FROM golang:1.27.4-alpine3.24 AS migration-build

RUN CGO_ENABLED=0 go install \
    -tags='no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb' \
    github.com/pressly/goose/v3/cmd/goose@v3.26.0

# Образ сервиса миграций
FROM alpine:3.24.4 AS migrate

COPY --from=migration-build /go/bin/goose /usr/local/bin/goose

WORKDIR /migrations

ENTRYPOINT ["goose"]

# Образ приложения
FROM alpine:3.24.4

WORKDIR /app

RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

COPY --from=build /screener ./screener

USER appuser:appgroup

EXPOSE 8080

CMD ["./screener"]