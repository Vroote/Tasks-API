# tasks-api

Небольшой REST API для списка задач на Go. Делал, чтобы разобраться с OpenAPI и подходом contract-first: сначала пишется спецификация `api/openapi.yaml`, а потом по ней генерируется код через [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen).

Что есть:
- `POST /api/v1/tasks` — создать задачу
- `GET /api/v1/tasks` — список задач (фильтр `status`, пагинация `limit`/`offset`)
- `GET /api/v1/tasks/{id}` — получить задачу по id
- `GET /docs` — Swagger UI

Запросы валидируются по спецификации (middleware из kin-openapi), а в тестах каждый ответ сервера проверяется на соответствие контракту. Данные хранятся в памяти.

## Запуск

```
go mod tidy
make run
```

Сервер стартует на `localhost:8080`, порт можно поменять через переменную `PORT`.

```
curl -X POST localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Сдать лабу", "dueDate": "2026-11-15"}'
```

Тесты: `make test`
