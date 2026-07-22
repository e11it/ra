# ra — Request Authenticator

`ra` — сервис авторизации и прокси перед Kafka REST Proxy.

Основные режимы:
- `auth_request` для nginx (`GET /auth`)
- proxy-режим (`/topics/*proxyPath`) c ACL и опциональной body validation

В режиме прокси может выполнять проверки тела сообщения.

## Эндпоинты

- `GET /auth` — проверка ACL и (для `POST`) проверка тела при включенной body validation.
- `ANY /topics/*proxyPath` — proxy в upstream при `proxy.enabled: true`.
- `GET /metrics` — Prometheus metrics.
- `GET /swagger/ra.yaml` — OpenAPI spec in YAML.
- `GET /swagger` — HTML docs (Swagger UI).
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

- Lint (pinned): `make go-lint-install && make go-lint` (`golangci-lint v2.11.4`)
- Fast local checks: `make qa-fast`
- Full local security profile: `make qa-security`
- Trivy filesystem scan: `make scan-fs`
- Trivy image scan: `make scan-image`
- DAST baseline (ZAP): `make dast`

Details: `docs/security.md`.

### Pre-commit hooks (локально)

В репозитории настроен `pre-commit` для базовых sanity-checks и `golangci-lint` (public + company).

Установка и включение:

```bash
python3 -m pip install --user pre-commit
pre-commit install
```

Запуск вручную:

```bash
pre-commit run -a
```

## Конфиг (ключевые блоки)

```yaml
trimurlprefix: /topics/

# Не писать access-лог (Gin) для перечисленных path (без query); по умолчанию /metrics, Swagger, и т.д.
access_log:
  exclude_paths: [/metrics, /health, /ready, /swagger, /swagger/ra.yaml]

proxy:
  enabled: true
  proxyhost: "http://rest-proxy:8082"

identity:
  authenticated_user_header: X-Authenticated-User
  # Exact nginx IP (/32 or /128) or a narrow nginx network CIDR.
  trusted_proxies: [10.20.0.10/32]

body_validation:
  enabled: true
  allowed_operations: [CREATE, UPDATE, UPSERT, DELETE, SNAPSHOT, EVENT]
  checks: [no_partition, is_tombstone, envelope, payload, entity_key]
```

`GET /reload` применяет reloadable-поля атомарно. Поля `addr`,
`proxy.enabled`, `proxy.proxyhost` и `access_log.exclude_paths` являются
startup-only, поскольку HTTP server/router/middleware захватывают их при старте.
Если candidate меняет хотя бы одно из них, reload целиком отклоняется с ошибкой
`restart required`; активная конфигурация остаётся прежней. Чтобы применить эти
поля, перезапустите RA.

nginx должен аутентифицировать клиента и передавать нормализованное имя в
`X-Authenticated-User`. RA не использует Basic Auth как источник identity и
доверяет этому header только если фактический socket peer (`RemoteAddr`), а не
`X-Forwarded-For`, входит в `identity.trusted_proxies`. Прямой клиент или
недоверенный proxy получает identity `anon`. В production дополнительно закройте
сетевой доступ к RA так, чтобы к нему мог обращаться только nginx.

## Body validation (company build)

Валидация тела (`Kafka REST v2 produce`) активна только с build tag `company`.
Public build завершает startup/reload ошибкой, если `body_validation.enabled: true`.

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
  "message": "Ra: payload validation errors. Problems: [key_mismatch]. Trace ID: c7fd43f8-1357-4b75-92f8-8d9fd9566ddd.",
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


### DEV

Kafka REST mock server

```sh
python3 - <<'PY'
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        self.send_response(200)
        self.end_headers()
    def log_message(self, *args): pass
HTTPServer(("127.0.0.1", 9999), H).serve_forever()
PY
```
