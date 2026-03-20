# Weather (Go + PostgreSQL + OpenWeatherMap + DB cache)

Сервис на Go хранит настройки пользователя/устройства в PostgreSQL, ходит в OpenWeatherMap за погодой, кэширует результат в БД, чтобы не дергать внешний API слишком часто, и отдает данные фронтенду.

## Стек

- Go (net/http + gorilla/mux)
- PostgreSQL (pgxpool)
- OpenWeatherMap API
- Frontend: статические файлы в папке `frontend/`

## Запуск

### 1) Подготовьте окружение

1. Заполните файл `.env` в корне проекта:

   - `OPENWEATHERMAP_API_KEY` — ваш API ключ OpenWeatherMap
   - `WEATHER_CACHE_TTL_SECONDS` — TTL кэша (по умолчанию 600)
   - `DBHost`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `POSTGRES_DB`

   пример .env

   ```bash
   PORT=8080
   POSTGRES_DB="postgres"
   DBHost="localhost"
   DB_PORT=5433
   DB_USER="postgres"
   DB_PASSWORD="password"
   OPENWEATHERMAP_API_KEY=""
   OPENWEATHERMAP_BASE_URL="https://api.openweathermap.org"
   WEATHER_CACHE_TTL_SECONDS=600
   ```

````

### 2) Поднимите PostgreSQL

В корне проекта:

```bash
docker-compose up -d
````

### 3) Запустите Go-сервер

```bash
go run ./cmd/app.go
```

Сервер:

- слушает `PORT`
- раздает frontend на `/`
- предоставляет API эндпоинты `/create_user` и `/api/weather`

## Frontend

Откройте в браузере:

`http://localhost:8080/`

На странице:

- задайте `name` и `city` и нажмите кнопку “Создать устройство”
- по кнопке “Обновить погоду” будет выполняться `GET /api/weather`

`device_id` сохраняется в `localStorage`, поэтому повторные обновления идут с тем же устройством.

## API

### 1) Создание пользователя/устройства

`POST /create_user`

Тело запроса (JSON):

```json
{
  "name": "Иван",
  "city": "Москва"
}
```

Ответ (JSON):

```json
{
  "device_id": 1
}
```

### 2) Получение погоды (с кэшированием)

`GET /api/weather`

Идентификатор устройства можно передать одним из способов:

- query param: `/api/weather?device_id=1`
- или header: `X-Device-Id: 1`

Если `device_id` не передан — используется первое устройство из БД.

Ответ (JSON):

```json
{
  "city": "Москва",
  "temp": 5.2,
  "condition": "облачно",
  "from_cache": true
}
```

`from_cache` показывает, был ли ответ взят из `weather_cache`.

## Схема базы данных

Таблицы создаются при старте сервиса (миграция в коде).

Важно: таблица `"user"` создается с кавычками, потому что `user` — зарезервированное слово в PostgreSQL.

Таблицы:

- `"user"`

  - `id` BIGSERIAL PRIMARY KEY
  - `name` TEXT NOT NULL

- `device`

  - `id` BIGSERIAL PRIMARY KEY
  - `user_id` BIGINT NOT NULL REFERENCES `"user"(id)` ON DELETE CASCADE
  - `city` TEXT NOT NULL

- `weather_cache`
  - `device_id` BIGINT PRIMARY KEY REFERENCES `device`(id) ON DELETE CASCADE
  - `temp` DOUBLE PRECISION NOT NULL
  - `condition` TEXT NOT NULL
  - `fetched_at` TIMESTAMPTZ NOT NULL
  - `expires_at` TIMESTAMPTZ NOT NULL

## Проверка вручную через curl

Создать устройство:

```bash
curl -s -X POST http://localhost:8080/create_user \
  -H "Content-Type: application/json" \
  -d '{"name":"Иван","city":"Москва"}'
```

Получить погоду (используя device_id из ответа):

```bash
curl -s "http://localhost:8080/api/weather?device_id=1"
```
