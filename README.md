Добавить кеш проверок
https://pkg.go.dev/github.com/hashicorp/golang-lru



* Список acl проверяется сверху вниз, пока не найдется успешное правило
* если под url попадает несколько правил, то сработает первое успешное.
* если под url удовлетворяет нескольк правил, но все 

TODO:
- Мониторинг. Промахи в кеше

- Bench: https://e11it.github.io/ra/dev/bench/

config.yml
```
auth:
  prefix: string
  allow_content_type: regexp
  url_mask: regexp
  urls: name
    url_mask: regexp
    urls:
    - name: string

  acl:
  - mask: 000-0\.sap-erp.*
    users:
    - sap
    methods:
    - post
    content_type:
    - sdfasdfasdf
```

Debug RA
```
curl -v \
  -u kafka-enforce.prod.AstueStag@rest.kafka.prod:somepassword \
  -H "Content-Type: application/vnd.kafka.avro.v2+json" \
  -H "X-Original-Uri: /topics/006-0.kafka-enforce.db.ts.stagdok.ec.energy-meter.0" \
  -H "X-Original-Method: POST" \
  http://ra:8080/auth
```

## Проверка тела сообщения (Kafka REST v2 Avro)

Модуль `pkg/kafkarest` умеет валидировать тело POST-запроса к Kafka REST v2 (`/topics/<topic>`) по корпстандарту из `docs/Корпоративный стандарт сообщений Kafka на базе Avro.md`. Включается блоком `body_validation` в конфиге:

```yaml
body_validation:
  enabled: true
  allowed_operations: [CREATE, UPDATE, UPSERT, DELETE, SNAPSHOT, EVENT]
  checks: [entity_key_match, operation_allowed]
```

Встроенные чекеры:

- `entity_key_match` — проверяет `records[i].key == envelope.meta.entityKey`. Для `operation=EVENT` допускается пустое `entityKey` и отсутствие `key`.
- `operation_allowed` — проверяет, что `envelope.meta.operation` входит в `allowed_operations`.

Tombstone-записи (`{"key": "...", "value": null}`) распознаются автоматически и пропускаются без чекеров — по стандарту у них нет envelope.

Нарушение любого чекера → HTTP `400 Bad Request` c деталями в заголовке `X-RA-ERROR`.

### Как добавить свой чекер

1. В `pkg/kafkarest/check_<name>.go` реализуйте тип, удовлетворяющий `RecordChecker` (поля `Name() string` и `Check(CheckContext, *Record) error`).
2. Зарегистрируйте фабрику в `defaultRegistry` в `pkg/kafkarest/registry.go`.
3. Добавьте имя чекера в `checks:` конфига.
4. Покройте юнит-тестом в пакете `kafkarest`.

### Ограничения режима `auth_request`

В режиме `auth_request` (nginx-subrequest на `/auth`) тело запроса **не передаётся** в subrequest nginx'ом по умолчанию. Чтобы body validation имела смысл, есть два варианта:

- **Рекомендуется**: использовать встроенный proxy-режим ra (`proxy.enabled: true`, `proxy.proxyhost: http://rest-proxy:8082`) и маршрут `/topics/*proxyPath` — тело доступно напрямую и проксируется дальше после валидации.
- Либо настроить nginx так, чтобы тело попадало в subrequest — например, через OpenResty/lua или директиву `mirror`. В конфиге примера `example/nginx/conf.d/secured.conf` по умолчанию стоит `proxy_pass_request_body off` — для включения body validation потребуется перенастройка.

// -H "Authorization: Basic bDMtb3JhLXB0czoxMjM="
updater!!!

https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#config-http-conn-man-headers-x-request-id