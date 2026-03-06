# kafkagen

CLI-утилита на Go, которая по Go-файлу с описанием интерфейсов генерирует Kafka-клиентов и сервер с роутингом сообщений.

Модель коммуникации — **fire-and-forget**: клиент отправляет сообщение в Kafka, сервер получает и вызывает соответствующий handler. Ответ не предусмотрен.

---

## Установка

```bash
git clone https://github.com/PIRSON21/generator-kafka-client
cd generator-kafka-client
go build -o kafkagen .
```

---

## Использование

```bash
kafkagen -schema=<path> -out=<dir> -package=<name>
```

| Флаг | Обязательный | Описание |
|------|:---:|---------|
| `-schema` | да | Путь к Go-файлу схемы |
| `-out` | да | Директория для сгенерированных файлов |
| `-package` | да | Имя Go-пакета в сгенерированных файлах |

Путь импорта Kafka-враппера вычисляется автоматически — генератор ищет `go.mod` выше директории `-out` и собирает путь `<module>/<rel-out>/kafka`.

---

## Формат схемы

Файл схемы — валидный Go-код с пакетом `schema`. Может содержать **один или несколько интерфейсов** — каждый становится отдельным клиентом. Сервер при этом один, принимает handler'ы для каждого интерфейса.

Сигнатура каждого метода:

```go
MethodName(ctx context.Context, req MethodNameRequest) error
```

- первый аргумент — `context.Context`
- второй аргумент — структура, объявленная в том же файле
- возвращаемое значение — только `error`

Если поля структур не имеют тегов — автоматически генерируется `json:"snake_case"`. Существующие теги копируются как есть.

---

## Что генерируется

Для схемы с двумя интерфейсами `Contract` и `Tariff` генератор создаёт:

```
out/
  types.go       — EventType-константы и структуры запросов
  client.go      — ContractClient + TariffClient (по одному на интерфейс)
  server.go      — ContractHandler + TariffHandler + единый NewServer(...)
  kafka/
    config.go    — ConfigProducer, ConfigConsumer
    producer.go  — реализация Producer
    consumer.go  — реализация Consumer
```

### types.go

```go
const (
    EventAddContract EventType = "add_contract"
    EventAddTariff   EventType = "add_tariff"
)
```

### client.go

```go
// По одному клиенту на каждый интерфейс схемы
func NewContractClient(cfg ClientConfig) (ContractClient, error)
func NewTariffClient(cfg ClientConfig)   (TariffClient, error)
```

### server.go

```go
// Один сервер, принимает handler'ы для всех интерфейсов
func NewServer(cfg ServerConfig, contract ContractHandler, tariff TariffHandler) (*serverRouter, error)

func (s *serverRouter) Run(ctx context.Context) error
func (s *serverRouter) Close() error
```

---

## Протокол сообщений

Каждое сообщение в Kafka — JSON:

```json
{
  "event_type": "add_contract",
  "data": { "contract_id": 42 }
}
```

Ключ сообщения: `time.Now().Format(time.RFC3339Nano)`.
Один топик на весь сервис — задаётся в `ConfigProducer.Topic` / `ConfigConsumer.Topic`.

---

## Тестовый запуск

В репозитории есть тестовая схема `testdata/billing/schema.go`:

```go
package schema

import "context"

type AddContractRequest struct {
    ContractID int
}

type AddTariffRequest struct {
    TariffID int
    Name     string
}

type Contract interface {
    AddContract(ctx context.Context, req AddContractRequest) error
}

type Tariff interface {
    AddTariff(ctx context.Context, req AddTariffRequest) error
}
```

Запуск:

```bash
# Сборка
go build -o kafkagen .

# Генерация
./kafkagen \
  -schema=./testdata/billing/schema.go \
  -out=./out \
  -package=billingclient
```

Ожидаемый вывод:

```
Generated in ./out:
  types.go
  client.go
  server.go
  kafka/
```

Проверить сгенерированный импорт:

```bash
head -10 out/client.go
# kafkawrapper "github.com/PIRSON21/generator-kafka-client/out/kafka"
```

### Пример использования сгенерированного кода

```go
// Клиент
client, err := billingclient.NewContractClient(billingclient.ClientConfig{
    Kafka: &billingclient.ConfigProducer{
        BootstrapServers: "localhost:9092",
        Topic:            "billing",
        Acks:             "all",
        ClientID:         "billing-producer",
    },
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()

err = client.AddContract(ctx, billingclient.AddContractRequest{ContractID: 42})

// Сервер
type contractHandler struct{}

func (h *contractHandler) AddContract(ctx context.Context, req *billingclient.AddContractRequest) error {
    log.Printf("contract: %+v", req)
    return nil
}

srv, err := billingclient.NewServer(
    billingclient.ServerConfig{
        Kafka: &billingclient.ConfigConsumer{
            BootstrapServers: "localhost:9092",
            Topic:            "billing",
            GroupID:          "billing-consumer",
            AutoOffsetReset:  "earliest",
        },
    },
    &contractHandler{},
    &tariffHandler{},
)
if err != nil {
    log.Fatal(err)
}
defer srv.Close()

if err := srv.Run(ctx); err != nil {
    log.Fatal(err)
}
```

---

## Тесты

```bash
# Все тесты
go test ./...

# Только парсер
go test ./internal/parser/...

# Только snake_case конвертер
go test ./internal/strconv/...

# Конкретный тест
go test ./internal/parser/... -run TestParse_TwoInterfaces
```

---

## Ошибки при валидации схемы

Генератор завершается с ненулевым exit code и понятным сообщением:

| Ситуация | Сообщение |
|----------|-----------|
| Файл не найден | `schema file not found` |
| Пакет не `schema` | `package must be "schema", got "..."` |
| Нет интерфейсов | `no interface found in schema` |
| Неверная сигнатура метода | `method "X": first param must be context.Context` |
| Отсутствует структура-аргумент | `struct "XRequest" not found for method "X"` |

---

## Структура проекта

```
.
├── main.go                          — CLI, точка входа
├── internal/
│   ├── model/model.go               — ServiceDef, InterfaceDef, MethodDef
│   ├── parser/parser.go             — парсинг schema.go через go/ast
│   ├── strconv/snake.go             — CamelCase → snake_case
│   └── generator/
│       ├── generator.go             — рендеринг шаблонов + копирование kafka/
│       ├── templates/               — types.go.tmpl, client.go.tmpl, server.go.tmpl
│       └── kafkasrc/                — исходники Kafka-враппера (embedded, копируются в out/kafka/)
├── kafka/                           — рабочая реализация враппера (для разработки генератора)
└── testdata/billing/schema.go       — пример схемы для тестового запуска
```
