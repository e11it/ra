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

// -H "Authorization: Basic bDMtb3JhLXB0czoxMjM="
updater!!!

https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#config-http-conn-man-headers-x-request-id