# ra — Request Authenticator

`ra` — сервис авторизации и прокси перед Kafka REST Proxy.

Основные режимы:
- `auth_request` для nginx (`GET /auth`)
- proxy-режим (`/topics/*proxyPath`) c ACL и опциональной body validation

## Эндпоинты

- `GET /auth` — проверка ACL и (для `POST`) проверка тела при включенной body validation.
- `ANY /topics/*proxyPath` — proxy в upstream при `proxy.enabled: true`.
- `GET /metrics` — Prometheus metrics.
- `GET /api/openapi/ra.yaml` — OpenAPI spec in YAML.
- `GET /api/openapi` — HTML docs (Swagger UI).
- `GET /reload` — reload конфига.

## OpenAPI generation

- Конфиг генерации: `api/oapi-config.yaml`
- Спека: `api/openapi/ra.yaml`
- Сгенерированный код (`models + embedded-spec`): `api/openapi/openapi.gen.go`

Команда:

```bash
make openapi-gen
```

## Quality and Security

- Fast local checks: `make qa-fast`
- Full local security profile: `make qa-security`
- Trivy filesystem scan: `make scan-fs`
- Trivy image scan: `make scan-image`
- DAST baseline (ZAP): `make dast`

Details: `docs/security.md`.

## Конфиг (ключевые блоки)

```yaml
trimurlprefix: /topics/

# Не писать access-лог (Gin) для перечисленных path (без query); по умолчанию /metrics, OpenAPI, и т.д.
access_log:
  exclude_paths: [/metrics, /health, /ready, /api/openapi, /api/openapi/ra.yaml]

proxy:
  enabled: true
  proxyhost: "http://rest-proxy:8082"

body_validation:
  enabled: true
  allowed_operations: [CREATE, UPDATE, UPSERT, DELETE, SNAPSHOT, EVENT]
  checks: [no_partition, is_tombstone, envelope, payload, entity_key]
```

## Body validation (company build)

Валидация тела (`Kafka REST v2 produce`) активна только с build tag `company`.

- `pkg/validate` — контракты/check-pipeline (`Report/Issue/Control`)
- `pkg/payloadvalidate` — protocol-layer parser для `records[]`
- `pkg/validate/common` — общие чекеры (сейчас `is_tombstone`)
- `pkg/validate/company` — корпоративные чекеры (`no_partition`, `envelope`, `payload`, `entity_key`)

Подробнее: `docs/packages.md`.

## Сборка

- Public: `make go-build` (`GO_TAGS=nomsgpack`)
- Company: `GO_TAGS="nomsgpack,company" make go-build`
- Docker public: `docker build -f docker/RA.Dockerfile .`
- Docker company: `docker build -f docker/RA.Dockerfile --build-arg GO_TAGS="nomsgpack,company" .`

## Локальный запуск примера

- `make docker-up RA_DOCKER_VARIANT=public`
- `make docker-up RA_DOCKER_VARIANT=company`

Compose-файл: `example/docker-compose.yml`.

## Наблюдаемость

- JSON access logs (`method`, `path`, `status`, `latency`, `x_request_id`, `x_ra_error`)
- Prometheus метрики:
  - `ra_http_requests_total`
  - `ra_http_request_duration_seconds`
  - `ra_http_inflight_requests`
  - `ra_auth_denied_total`
  - `ra_body_validation_failed_total`
  - `ra_proxy_upstream_errors_total`
  - `ra_config_reload_total{result}`

## Формат ошибок

RA возвращает ошибки в JSON-формате (включая `auth_request`):

```json
{
  "error_code": 42230,
  "message": "Ra: payload validation errors",
  "details": {
    "trace_id": "c7fd43f8-1357-4b75-92f8-8d9fd9566ddd",
    "errors": [
      {
        "record_index": 0,
        "path": "records[0].key",
        "code": "key_mismatch",
        "message": "record key \"554123\" does not match envelope.meta.entityKey \"5541s23\""
      }
    ]
  }
}
```

`X-RA-ERROR` остаётся в ответе как краткий summary для совместимости.

### Карта сценариев

- `auth deny` → HTTP `403`, `error_code: 40301`
- `body read / malformed body` → HTTP `400`, `error_code: 40010`
- `payload validation failed` → HTTP `422`, `error_code: 42230`
- `reload failed` → HTTP `400`, `error_code: 40020`

Каноничный список кодов, их природа и значение хранится в `internal/app/ra/error_codes.go`.
