package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/mr-tron/base58"
)

// Keypair структура для загрузки ключа из JSON файла
type Keypair struct {
	PrivateKey []uint8 `json:"privateKey"`
}

// loadKeypairFromFile загружает приватный ключ из JSON файла
func loadKeypairFromFile(path string) (solana.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	var keypair Keypair
	if err := json.Unmarshal(data, &keypair); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if len(keypair.PrivateKey) != 64 {
		return nil, fmt.Errorf("неверный формат приватного ключа: ожидается 64 байта, получено %d", len(keypair.PrivateKey))
	}

	// Конвертируем []uint8 в []byte
	privateKeyBytes := make([]byte, 64)
	for i, v := range keypair.PrivateKey {
		privateKeyBytes[i] = byte(v)
	}

	return solana.PrivateKey(privateKeyBytes), nil
}

// createTransferInstruction создает инструкцию для перевода SOL
func createTransferInstruction(from, to solana.PublicKey, amount uint64) solana.Instruction {
	// System Program ID
	systemProgramID := solana.SystemProgramID

	// Создаем данные инструкции: 4 байта для типа инструкции (2 = Transfer) + 8 байт для суммы
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:4], 2) // Transfer instruction type
	binary.LittleEndian.PutUint64(data[4:12], amount)

	// Создаем инструкцию используя solana.NewInstruction
	instruction := solana.NewInstruction(
		systemProgramID,
		solana.AccountMetaSlice{
			{PublicKey: from, IsWritable: true, IsSigner: true},
			{PublicKey: to, IsWritable: true, IsSigner: false},
		},
		data,
	)

	return instruction
}

// sendSOL отправляет SOL с одного адреса на другой
func sendSOL(ctx context.Context, from solana.PrivateKey, to solana.PublicKey, amount uint64, rpcClient *rpc.Client) (solana.Signature, error) {
	// Получаем публичный ключ отправителя
	fromPubkey := from.PublicKey()

	// Получаем последний blockhash
	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("ошибка получения blockhash: %w", err)
	}

	// Создаем инструкцию перевода SOL
	instruction := createTransferInstruction(fromPubkey, to, amount)

	// Создаем транзакцию
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(fromPubkey),
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("ошибка создания транзакции: %w", err)
	}

	// Подписываем транзакцию
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if key.Equals(fromPubkey) {
				return &from
			}
			return nil
		},
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("ошибка подписания транзакции: %w", err)
	}

	// Отправляем транзакцию
	sig, err := rpcClient.SendTransaction(ctx, tx)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("ошибка отправки транзакции: %w", err)
	}

	return sig, nil
}

// waitForConfirmation ожидает подтверждения транзакции
func waitForConfirmation(ctx context.Context, signature solana.Signature, rpcClient *rpc.Client, wsURL string) error {
	// Подключаемся к WebSocket для отслеживания статуса
	wsClient, err := ws.Connect(ctx, wsURL)
	if err != nil {
		return fmt.Errorf("ошибка подключения к WebSocket: %w", err)
	}
	defer wsClient.Close()

	// Подписываемся на обновления статуса транзакции
	sub, err := wsClient.SignatureSubscribe(
		signature,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return fmt.Errorf("ошибка подписки на статус: %w", err)
	}
	defer sub.Unsubscribe()

	// Ждем подтверждения с таймаутом
	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("таймаут ожидания подтверждения транзакции")
		case resp, ok := <-sub.Response():
			if !ok {
				return fmt.Errorf("канал закрыт")
			}
			if resp.Value.Err != nil {
				return fmt.Errorf("ошибка транзакции: %v", resp.Value.Err)
			}
			// Транзакция подтверждена
			return nil
		}
	}
}

// convertKeyFromBase58 конвертирует приватный ключ из base58 в JSON формат
func convertKeyFromBase58(base58Key string, outputFile string) error {
	// Декодируем base58 строку в байты
	decodedBytes, err := base58.Decode(base58Key)
	if err != nil {
		return fmt.Errorf("ошибка декодирования base58: %w", err)
	}

	// Проверяем длину - должен быть полный keypair (64 байта)
	if len(decodedBytes) != 64 {
		return fmt.Errorf("неверная длина ключа: ожидается 64 байта (полный keypair), получено %d", len(decodedBytes))
	}

	// Полный keypair (64 байта: 32 приватных + 32 публичных)
	fullKeypair := make([]uint8, 64)
	copy(fullKeypair, decodedBytes)

	// Создаем приватный ключ из полного keypair
	privateKey := solana.PrivateKey(decodedBytes)
	fmt.Println("Private key (base58):", privateKey.String())
	fmt.Println("Public key:", privateKey.PublicKey().String())

	// Создаем структуру для JSON
	keypair := Keypair{
		PrivateKey: fullKeypair,
	}

	// Сериализуем в JSON
	jsonData, err := json.MarshalIndent(keypair, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка создания JSON: %w", err)
	}

	// Записываем в указанный файл
	if err := os.WriteFile(outputFile, jsonData, 0600); err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	fmt.Printf("\n Ключ успешно конвертирован и сохранен в: %s\n", outputFile)

	// Автоматически создаем phantom-keypair.json (если это не тот же файл)
	if outputFile != "phantom-keypair.json" {
		if err := os.WriteFile("phantom-keypair.json", jsonData, 0600); err != nil {
			return fmt.Errorf("ошибка записи phantom-keypair.json: %w", err)
		}
		fmt.Printf(" Автоматически создан файл: phantom-keypair.json\n")
	}
	return nil
}

func main() {
	// Парсинг аргументов командной строки
	var (
		keypairPath   = flag.String("keypair", "", "Путь к файлу с приватным ключом (JSON формат)")
		recipient     = flag.String("to", "", "Адрес получателя (публичный ключ)")
		amount        = flag.Float64("amount", 0, "Сумма для отправки в SOL")
		convertKey    = flag.String("convert-key", "", "Конвертировать приватный ключ из base58 в JSON формат")
		convertOutput = flag.String("convert-output", "phantom-keypair.json", "Имя выходного файла для конвертации (используется с -convert-key, по умолчанию: phantom-keypair.json)")
		rpcURL        = flag.String("rpc-url", rpc.DevNet_RPC, "URL RPC эндпоинта (по умолчанию: devnet, можно указать QuickNode)")
		wsURL         = flag.String("ws-url", rpc.DevNet_WS, "URL WebSocket эндпоинта (по умолчанию: devnet, можно указать QuickNode)")
	)
	flag.Parse()

	// Режим конвертации ключа
	if *convertKey != "" {
		if err := convertKeyFromBase58(*convertKey, *convertOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка конвертации: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Валидация аргументов для режима отправки
	if *keypairPath == "" {
		fmt.Fprintf(os.Stderr, "Ошибка: необходимо указать путь к файлу с ключом через -keypair\n")
		fmt.Fprintf(os.Stderr, "Или используйте -convert-key для конвертации ключа из base58\n")
		flag.Usage()
		os.Exit(1)
	}

	if *recipient == "" {
		fmt.Fprintf(os.Stderr, "Ошибка: необходимо указать адрес получателя через -to\n")
		flag.Usage()
		os.Exit(1)
	}

	if *amount <= 0 {
		fmt.Fprintf(os.Stderr, "Ошибка: сумма должна быть больше 0\n")
		flag.Usage()
		os.Exit(1)
	}

	// Парсинг адреса получателя
	recipientPubkey, err := solana.PublicKeyFromBase58(*recipient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: неверный адрес получателя: %v\n", err)
		os.Exit(1)
	}

	// Загрузка приватного ключа
	privateKey, err := loadKeypairFromFile(*keypairPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки ключа: %v\n", err)
		os.Exit(1)
	}

	fromPubkey := privateKey.PublicKey()
	fmt.Printf("Отправитель: %s\n", fromPubkey.String())
	fmt.Printf("Получатель: %s\n", recipientPubkey.String())

	// Создание контекста
	ctx := context.Background()

	// Создание RPC клиента (используем кастомный URL или devnet по умолчанию)
	rpcClient := rpc.New(*rpcURL)

	// Проверка баланса отправителя
	balance, err := rpcClient.GetBalance(ctx, fromPubkey, rpc.CommitmentFinalized)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка получения баланса: %v\n", err)
		os.Exit(1)
	}

	balanceSOL := float64(balance.Value) / 1e9
	fmt.Printf("Баланс отправителя: %.9f SOL\n", balanceSOL)

	// Конвертируем SOL в lamports
	amountLamports := uint64(*amount * 1e9)

	// Проверяем, достаточно ли средств (с учетом комиссии)
	requiredAmount := amountLamports + 5000 // Примерная комиссия
	if balance.Value < requiredAmount {
		fmt.Fprintf(os.Stderr, "Ошибка: недостаточно средств. Требуется: %.9f SOL, доступно: %.9f SOL\n",
			float64(requiredAmount)/1e9, balanceSOL)
		os.Exit(1)
	}

	// Отправка SOL
	fmt.Printf("Отправка %.9f SOL...\n", *amount)
	signature, err := sendSOL(ctx, privateKey, recipientPubkey, amountLamports, rpcClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка отправки: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Транзакция отправлена. Подпись: %s\n", signature.String())
	fmt.Printf("Ожидание подтверждения...\n")

	// Ожидание подтверждения
	if err := waitForConfirmation(ctx, signature, rpcClient, *wsURL); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка подтверждения: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Транзакция успешно подтверждена!\n")
	fmt.Printf("Просмотр транзакции: https://explorer.solana.com/tx/%s?cluster=devnet\n", signature.String())
}
