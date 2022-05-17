Добавить кеш проверок
https://pkg.go.dev/github.com/hashicorp/golang-lru



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