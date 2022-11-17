Добавить кеш проверок
https://pkg.go.dev/github.com/hashicorp/golang-lru



* Список acl проверяется снизу вверх(чтобы можно было дописывать правила в конец)
* Отказ возвразается если совпаюает url, username, method

TODO:
- Мониторинг. Промахи в кеше

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