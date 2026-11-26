# Screener_backend

## Требования

- Docker с поддержкой Linux-контейнеров
- Docker Compose v2
- Make

Go для запуска через Docker устанавливать не требуется.

## Запуск

Скопируйте пример настроек:

```bash
cp .env.example .env
```

Запустите окружение:

```bash
make up
```

Команда соберёт образ приложения, запустит PostgreSQL и приложение.
Приложение запускается после успешной проверки готовности PostgreSQL.

Проверить состояние контейнеров:

```bash
docker compose ps
```

Приложение доступно по адресу [http://localhost:8080/api/v1/](http://localhost:8080/api/v1/).
Ожидаемый ответ:

```json
{"status":"ok"}
```

## Настройки

Настройки задаются в `.env`. Пример находится в `.env.example`.
Файл `.env` не добавляется в Git.

`DATABASE_URL` формируется Docker Compose из настроек `POSTGRES_USER`,
`POSTGRES_PASSWORD` и `POSTGRES_DB`. Они определены выше строки подключения
в файле настроек. Для значений со специальными символами URL может потребоваться
URL-кодирование при составлении строки подключения.

PostgreSQL доступен приложению по адресу `postgres:5432`, а с компьютера —
по адресу `localhost:${POSTGRES_PORT}`. `POSTGRES_PORT` задаёт порт компьютера;
порт PostgreSQL внутри контейнера остаётся `5432`.

## Остановка

```bash
make down
```

Данные PostgreSQL сохраняются в named volume и доступны после повторного `make up`.

## Просмотр логов

```bash
docker compose logs -f app
docker compose logs -f postgres
```

Для выхода из просмотра логов нажмите Ctrl+C.
