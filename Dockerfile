# Этап 1: сборка приложения
FROM golang:1.27.4-alpine3.24 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /screener ./cmd/screener


# Этап 2: запуск приложения
FROM alpine:3.24.4

WORKDIR /app

RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

COPY --from=build /screener ./screener

USER appuser:appgroup

EXPOSE 8080

CMD ["./screener"]