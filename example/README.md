# Пример Kafka REST с RA за nginx

В примерах используется мое/наше соглашение об именовании.

В `config.yml` описаны проверки:
- имя топика должно удолвлетворять re(всегда): `^\d{3}-\d(-\d{3}-\d)?\.[a-z0-9-]+\.(db|cdc|cmd|sys|log|tmp)\.[a-z0-9-.]+\.\d+$`
- пользователю `avro-test` разрешено писать(POST) в топики `^888-8\.example\.db\.` и в формате(contenttype): `application/vnd.kafka.avro.v2+json`.

1. Проверяем, что если все условния выполняются, то пользователь может записать:


```sh
curl -u avro-user:anypassword -X POST \
      -H "Content-Type: application/vnd.kafka.avro.v2+json" \
      -H "Accept: application/vnd.kafka.v2+json" \
      -H "X-Request-ID: 5f385ffc-2419-44cd-8ae5-322f0b3b6856" \
      --data '{"value_schema": "{\"type\": \"record\", \"name\": \"User\", \"fields\": [{\"name\": \"name\", \"type\": \"string\"}]}", "records": [{"value": {"name": "testUser"}}]}' \
      "http://localhost:8080/topics/888-8.example.db.awesome.0"
```

Ответ:
```json
{"offsets":[{"partition":0,"offset":0,"error_code":null,"error":null}],"key_schema_id":null,"value_schema_id":1
```

В логе RA будет отметка:
```
 time="2022-05-17T17:41:31Z" level=info msg="{888-8.wrong-group.db.awesome.0 avro-user 172.21.0.1 POST application/vnd.kafka.avro.v2+json}"
```

2. Отказ и возврат ошибки

В случае, если `RA` не разрешает дальнейшее выполнение, пользователю возвращается код `403`.

```
<html>
<head><title>403 Forbidden</title></head>
<body>
<center><h1>403 Forbidden</h1></center>
<hr><center>nginx/1.16.0</center>
</body>
</html>
```

Пример лога с ошибкой(пользователю avro-user не разрешено писать в топики `888-8.wrong-group.db.*`)
```
example-ra-1                       | time="2022-05-17T17:41:31Z" level=error ContentType=application/vnd.kafka.avro.v2+json IP=172.21.0.1 Method=POST URL=888-8.wrong-group.db.awesome.0 User=avro-user error="Permission denied"
example-ra-1                       | time="2022-05-17T17:41:31Z" level=error msg="Error #01: Permission denied\n" clientIP=172.21.0.1 dataLength=0 hostname=defe4e65fa9e latency=0.265606 method=GET path=/auth referer= statusCode=403 userAgent=curl/7.79.1
```


Reload `ra` config:

```
docker-compose kill -s SIGHUP ra
```