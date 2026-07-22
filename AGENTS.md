# AGENTS.md — заметки для ИИ-агентов и разработчиков

Этот файл — короткая сводка правил и контекста для любого агента (Cursor, Codex, Claude Code и т.п.), работающего в репозитории.

## Что за проект

`ra` — Request Authenticator: HTTP-сервис для nginx `auth_request` и прокси-режима перед Kafka REST Proxy. Проверяет URL, ACL, опционально — тело сообщения Kafka REST v2 (Avro-envelope по корпстандарту).

Подробный контекст и стиль — в `.cursor/rules/`:

- `.cursor/rules/project.mdc` — обзор проекта, стек, точки входа, верификационные команды.
- `.cursor/rules/go-style.mdc` — соглашения по Go-коду.
- `.cursor/rules/kafka-avro-standard.mdc` — инварианты корпстандарта и коду в `pkg/validate` / `pkg/validate/common` / `pkg/validate/company` / `pkg/payloadvalidate`.
- `docs/packages.md` — зачем отдельно `pkg/validate`, `pkg/payloadvalidate`, `pkg/validate/common` и `pkg/validate/company`.

Все три файла применяются автоматически (первый — всегда, остальные — по glob).

## Канонический рабочий цикл

1. Прочитать релевантные rules (если они не подгружены автоматически).
2. Внести изменения, соблюдая модульность (`pkg/<name>/interface.go` + реализации) и стиль.
3. Добавить / обновить тесты.
4. Прогнать:
   ```bash
   go vet ./...
   go test ./...
   go build ./...
   ```
5. Обновить документацию (README, `docs/`, примеры в `example/`), если меняется контракт или конфигурация.

## Ключевые документы

- `docs/Корпоративный стандарт сообщений Kafka на базе Avro.md` — источник истины по формату сообщений (envelope, operation, entityKey, tombstone).
- `docs/packages.md` — смысл пакетов валидации.
- `README.md` — короткое описание проекта и запуск примеров.

## Версии и окружение

- Go `1.26.3` (Docker-образы в `docker/*.Dockerfile`).
- Запуск контейнеров — через `Makefile` (`make docker-up`, `make docker-down`, и т.д.).

## Learned User Preferences

- Корпоративную логику держать вне публичных пакетов: envelope/типы/чекеры корп-стандарта не оставлять в `pkg/payloadvalidate` и `pkg/validate`, а переносить в `pkg/validate/company`.
- Не добавлять в конфиг проверки, которые уже закрыты выше по стеку (размер тела — nginx/Kafka REST; content-type — существующие per-topic правила). Если Kafka REST уже валидирует Avro-схему/типы, в RA оставлять только бизнес-правила и не дублировать value-checks (например для `businessDate`/`businessKey`/`sequence`).
- Proxy-режим должен быть best-effort: при ошибке декодирования ответа upstream передавать исходный статус/тело клиенту, а не возвращать 500 и не обрывать соединение.
- В публичных пакетах не тянуть backward-compat: при миграциях удалять неиспользуемые типы и переходные обёртки.
- Один multi-stage `docker/RA.Dockerfile` с выбором варианта сборки через build tag/ENV предпочтительнее отдельных Dockerfile на public/company.
- Валидаторы тела должны агрегировать ошибки (возвращать массив), а не fail-fast, чтобы пользователь правил всё за один проход.
- Проверки обязаны валидировать JSON-тип значения, а не только его наличие: в поле может прийти любой валидный JSON (число вместо строки и т.п.).
- Чекеры должны быть композируемыми: родительский чекер сам вызывает подчекеры (например, внутри `meta`), без опоры на порядок и без повторной сериализации payload.
- Ошибки API для validation-path должны возвращаться структурированным JSON-контрактом (`error_code`, `message`, `details`) вместо `text/plain`-ответов.
- `gitleaks` держать в pre-commit/pre-push, а не отдельной CI job.
- Security-проверки не должны сканировать локальные Go-кеши (`.gomodcache`, `.gocache`); анализировать только пакеты проекта.
- Для Go Docker-образа предпочтителен минимальный static/nonroot runtime с `CGO_ENABLED=0`, если CGO не нужен.

## Learned Workspace Facts

- Валидация тела Kafka REST v2 включается build tag `company`: без тега корп-чекеры не регистрируются и тело не проверяется.
- `pkg/validate/` — только публичные контракты чекеров и реестр; корпоративные проверки (envelope, `entityKey`, `operation`, `eventTimeZone`) живут в `pkg/validate/company/` за build tag `company`.
- `pkg/payloadvalidate/` — protocol-layer: парсит produce-запросы Kafka REST v2 и прогоняет зарегистрированные чекеры; корп-envelope и корп-специфика сюда не попадают.
- Tombstone-сценарий реализован отдельным generic-чекером `is_tombstone` в `pkg/validate/common`: он останавливает pipeline записи со статусом success.
- По корпстандарту поля envelope: `eventTimeZone` — обязательное строковое (tombstone — исключение), `entityKey` — обязательное строковое (не union), `schemaVersion` — обязательное целое и всегда `0`.
- Вариант Docker-сборки (public/company) выбирается через build tag/ENV в едином `docker/RA.Dockerfile`; отдельный `docker/Modules.Dockerfile` удалён.
- Канонический реестр кодов ошибок RA хранится в `internal/app/ra/error_codes.go` и используется для JSON error contract.
- OpenAPI поддерживается в `api/openapi/`: `oapi-codegen` генерирует `openapi.gen.go`, а сервер отдает YAML и HTML-документацию.
- Access-log в Gin поддерживает конфигурируемые исключения технических endpoint через `exclude_paths`; пустой список в YAML переопределяет дефолты configor.
- Security-профиль использует `gosec`, `govulncheck` и `gitleaks`; для `gosec` нужны исключения `.gomodcache`/`.gocache`, чтобы не обходить локальные кеши.
- Docker publish для git tags должен использовать GitHub Environment `docker`: username из `vars.DOCKER_USERNAME`, token из `secrets.DOCKERHUB_TOKEN`, образ `e11it/ra:<tag>`.
- В режиме `extendedAvroTypes` strict-union значения ожидаются в tagged-виде (`{"string":...}`, `{"date":...}`), при этом `eventTime`/`producedAt` остаются не-union полями и валидируются как обычные timestamp-строки.
