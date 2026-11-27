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

## Мониторинг

После `make up` откройте [Prometheus](http://localhost:9090). На странице
[Targets](http://localhost:9090/targets) цель `app` должна показывать `UP`.
Prometheus опрашивает `http://app:9090/metrics` в сети Compose каждые 15 секунд.
Приложению нужен `METRICS_ADDR=:9090` в `.env`. Том `prometheus_data` хранит историю за 7 дней.

### Запросы в PromQL

Среднее число HTTP-запросов в секунду за последние 5 минут:

```promql
sum(rate(screener_http_requests_total{job="app"}[5m]))
```

Процент ответов 5xx за последние 5 минут (при нулевом трафике доля не определена):

```promql
100 * (sum(rate(screener_http_requests_total{job="app",status=~"5.."}[5m])) or vector(0))
  / sum(rate(screener_http_requests_total{job="app"}[5m]))
```

p95 длительности HTTP-запросов в секундах за последние 5 минут:

```promql
histogram_quantile(0.95, sum by (le) (rate(screener_http_request_duration_seconds_bucket{job="app"}[5m])))
```

### Проверка алертов

Остановите приложение командой `docker compose stop app`. На странице
[Alerts](http://localhost:9090/alerts) правило `ScreenerAppDown` перейдёт в
`Firing` после минуты непрерывной недоступности. Выполните `docker compose start app`
и дождитесь восстановления цели. Правило `ScreenerHigh5xxRate` срабатывает, если
доля ответов 5xx за скользящее окно 5 минут превышает 5% в течение 2 минут.

Проверки из корня репозитория (для Git Bash на Windows предварительно выполните
`export MSYS_NO_PATHCONV=1`):

```bash
docker compose exec -T prometheus promtool check config /etc/prometheus/prometheus.yml
docker compose exec -T prometheus promtool check rules /etc/prometheus/alerts.yml
docker compose exec -T -w /etc/prometheus/tests prometheus promtool test rules alerts.test.yml
```
