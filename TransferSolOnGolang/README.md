# Скрипт для отправки SOL в Solana Devnet

Go приложение для отправки SOL транзакций в Solana devnet сети.

## Требования

- Go 1.25.1 или выше
- Аккаунт в Solana devnet с балансом SOL

## Установка

1. Клонируйте репозиторий или скачайте файлы проекта

2. Установите зависимости:
```bash
go mod download
```

3. Соберите проект:
```bash
go build -o transfer-sol main.go
```

## Подготовка ключа

Скрипт использует приватный ключ из JSON файла. Формат файла должен быть следующим:

```json
{
  "privateKey": [массив из 64 чисел от 0 до 255]
}
```

Пример файла `example-keypair.json` находится в корне проекта.

### Конвертация приватного ключа из base58

Скрипт поддерживает конвертацию приватного ключа из base58 формата (например, из Phantom кошелька) в JSON формат, который используется для отправки SOL.

**Использование:**
```bash
./transfer-sol -convert-key "ВАШ_ПРИВАТНЫЙ_КЛЮЧ_BASE58"
```

Или с указанием имени файла:
```bash
./transfer-sol -convert-key "ВАШ_ПРИВАТНЫЙ_КЛЮЧ_BASE58" -convert-output my-keypair.json
```

**Параметры:**
- `-convert-key` - приватный ключ в формате base58 (обязательный)
- `-convert-output` - имя выходного JSON файла (по умолчанию: `phantom-keypair.json`)

**Вывод:**
```
Private key (base58): [ваш приватный ключ]
Public key: [ваш публичный ключ]

Ключ успешно конвертирован и сохранен в: phantom-keypair.json
```

**Примечание:** Скрипт автоматически создает файл `phantom-keypair.json` при конвертации, даже если указан другой выходной файл.

## Использование

### Базовое использование

```bash
./transfer-sol -keypair <путь_к_файлу_ключа> -to <адрес_получателя> -amount <сумма_в_SOL>
```

### Параметры

- `-keypair` (обязательный) - путь к JSON файлу с приватным ключом
- `-to` (обязательный) - адрес получателя (публичный ключ в формате base58)
- `-amount` (обязательный) - сумма для отправки в SOL (например, 0.1 для 0.1 SOL)
- `-rpc-url` (опциональный) - URL RPC эндпоинта (по умолчанию: devnet, можно указать QuickNode)
- `-ws-url` (опциональный) - URL WebSocket эндпоинта (по умолчанию: devnet, можно указать QuickNode)

### Примеры

#### Отправка 0.1 SOL

```bash
./transfer-sol -keypair ./my-keypair.json -to 7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU -amount 0.1
```

#### Отправка 1 SOL

```bash
./transfer-sol -keypair ./my-keypair.json -to 7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU -amount 1.0
```

#### Использование с QuickNode

Если вы используете QuickNode для более быстрого и надежного подключения:

```bash
./transfer-sol \
  -keypair ./my-keypair.json \
  -to 7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU \
  -amount 0.1 \
  -rpc-url "https://your-endpoint.solana-devnet.quiknode.pro/YOUR_API_KEY/" \
  -ws-url "wss://your-endpoint.solana-devnet.quiknode.pro/YOUR_API_KEY/"
```

## Что делает скрипт

1. Загружает приватный ключ из JSON файла
2. Подключается к Solana devnet RPC (по умолчанию или через `-rpc-url`)
3. Проверяет баланс и создает транзакцию
4. Подписывает и отправляет транзакцию
5. Ожидает подтверждения (до 30 секунд)
6. Выводит ссылку на транзакцию в Solana Explorer

## Пример вывода

```
Отправитель: 7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU
Получатель: 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM
Баланс отправителя: 2.500000000 SOL
Отправка 0.100000000 SOL...
Транзакция отправлена. Подпись: 5VERv8NMvzbJMEkV8xnrLkEaWRtSz9CosKDYjCJjBRnbJLg8GdgyFKguNvfz9bJYj5v5qJjJjJjJjJjJjJjJjJj
Ожидание подтверждения...
Транзакция успешно подтверждена!
Просмотр транзакции: https://explorer.solana.com/tx/5VERv8NMvzbJMEkV8xnrLkEaWRtSz9CosKDYjCJjBRnbJLg8GdgyFKguNvfz9bJYj5v5qJjJjJjJjJjJjJjJjJj?cluster=devnet
```
