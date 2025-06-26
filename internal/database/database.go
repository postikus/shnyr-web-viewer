package database

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"shnyr/internal/config"
	"shnyr/internal/logger"
	"strconv"
	"strings"
	"sync"
)

// DatabaseManager содержит функции для работы с базой данных
type DatabaseManager struct {
	db     *sql.DB
	logger *logger.LoggerManager
	wg     sync.WaitGroup // для ожидания завершения асинхронных операций
}

// NewDatabaseManager создает новый экземпляр DatabaseManager
func NewDatabaseManager(db *sql.DB, loggerManager *logger.LoggerManager) *DatabaseManager {
	return &DatabaseManager{
		db:     db,
		logger: loggerManager,
	}
}

// SaveOCRResultToDB сохраняет результат OCR в базу данных
func (h *DatabaseManager) SaveOCRResultToDB(imagePath, ocrResult string, debugInfo, jsonData string, rawText string, imageData []byte, cfg *config.Config, itemCategory string, currentItemName string) (int, error) {
	// Проверяем настройку сохранения в БД
	if cfg.SaveToDB != 1 {
		h.logger.Info("Сохранение в БД отключено (save_to_db = %d)", cfg.SaveToDB)
		return 0, nil
	}
	// Создаем таблицу, если она не существует
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS ocr_results (
		id INT AUTO_INCREMENT PRIMARY KEY,
		image_path VARCHAR(255) NOT NULL,
		image_data LONGBLOB,
		ocr_text LONGTEXT,
		debug_info LONGTEXT,
		json_data LONGTEXT,
		raw_text LONGTEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := h.db.Exec(createTableSQL)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания таблицы: %v", err)
	}

	// Вставляем результат OCR с изображением
	insertSQL := `INSERT INTO ocr_results (image_path, image_data, ocr_text, debug_info, json_data, raw_text) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := h.db.Exec(insertSQL, imagePath, imageData, ocrResult, debugInfo, jsonData, rawText)
	if err != nil {
		return 0, fmt.Errorf("ошибка вставки данных: %v", err)
	}

	// Получаем ID вставленной записи
	ocrResultID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ошибка получения ID записи: %v", err)
	}

	h.logger.Info("✅ OCR результат сохранен с ID: %d", ocrResultID)

	// Сохраняем структурированные данные асинхронно
	if jsonData != "" {
		h.logger.Info("🔧 Запускаем асинхронное сохранение структурированных данных для OCR ID: %d", ocrResultID)

		// Запускаем асинхронное сохранение
		h.wg.Add(1)
		go func(ocrID int, jsonStr string) {
			defer h.wg.Done()
			err := SaveStructuredDataBatch(h.db, ocrID, jsonStr, itemCategory, currentItemName)
			if err != nil {
				h.logger.LogError(err, "Ошибка асинхронного сохранения структурированных данных")
			} else {
				h.logger.Info("✅ Структурированные данные успешно сохранены асинхронно для OCR ID: %d", ocrID)
			}
		}(int(ocrResultID), jsonData)
	} else {
		h.logger.Info("⚠️ JSON данные пустые, пропускаем сохранение structured items")
	}

	h.logger.Info("OCR результат и изображение сохранены в базу данных для файла: %s (ID: %d)", imagePath, ocrResultID)
	return int(ocrResultID), nil
}

// WaitForAsyncOperations ожидает завершения всех асинхронных операций сохранения
func (h *DatabaseManager) WaitForAsyncOperations() {
	h.logger.Info("⏳ Ожидаем завершения асинхронных операций сохранения...")
	h.wg.Wait()
	h.logger.Info("✅ Все асинхронные операции сохранения завершены")
}

// InitializeItemsTable создает таблицу предметов и инициализирует её из файла
func (h *DatabaseManager) InitializeItemsTable() error {
	h.logger.Info("🚀 Инициализация таблицы предметов...")

	// Пересоздаем таблицу с новой структурой
	err := h.RecreateItemsTable()
	if err != nil {
		return err
	}

	// Загружаем предметы из файла
	err = h.loadItemsFromFile("items.txt")
	if err != nil {
		return err
	}

	h.logger.Info("✅ Таблица предметов инициализирована успешно")
	return nil
}

// loadItemsFromFile загружает предметы из текстового файла
func (h *DatabaseManager) loadItemsFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Начинаем транзакцию
	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer tx.Rollback()

	// Подготавливаем запрос
	stmt, err := tx.Prepare("INSERT IGNORE INTO items_list (name, category, min_price) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса: %v", err)
	}
	defer stmt.Close()

	lineNumber := 0
	currentCategory := "buy_consumables" // По умолчанию первая категория - покупка расходников
	buyConsumablesCount := 0
	buyEquipmentCount := 0
	sellConsumablesCount := 0
	sellEquipmentCount := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Проверяем разделители
		switch line {
		case "---":
			// Определяем следующую категорию на основе текущей
			switch currentCategory {
			case "buy_consumables":
				currentCategory = "buy_equipment"
			case "buy_equipment":
				// После buy_equipment переходим к sell_consumables
				currentCategory = "sell_consumables"
			case "sell_consumables":
				currentCategory = "sell_equipment"
			case "sell_equipment":
				// Если уже в sell_equipment, остаемся там
				currentCategory = "sell_equipment"
			default:
				// Если это первый разделитель, переходим к buy_equipment
				currentCategory = "buy_equipment"
			}
			h.logger.Info("📋 Переключаемся на категорию: %s", currentCategory)
			continue
		case "===":
			// Принудительно переходим к sell_consumables
			currentCategory = "sell_consumables"
			h.logger.Info("📋 Переключаемся на категорию: %s", currentCategory)
			h.logger.Info("🔍 DEBUG: Найдем разделитель ===, переключаемся на sell_consumables")
			continue
		}

		// Парсим строку в формате "название:цена"
		parts := strings.Split(line, ":")
		itemName := strings.TrimSpace(parts[0])
		var minPrice float64 = 0

		if len(parts) > 1 {
			priceStr := strings.TrimSpace(parts[1])
			if priceStr != "" {
				if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
					minPrice = price
				} else {
					h.logger.LogError(fmt.Errorf("ошибка парсинга цены '%s' для предмета '%s' на строке %d", priceStr, itemName, lineNumber), "")
					// Продолжаем с ценой 0
				}
			}
		}

		// Вставляем предмет с категорией и минимальной ценой
		_, err := stmt.Exec(itemName, currentCategory, minPrice)
		if err != nil {
			return fmt.Errorf("ошибка вставки предмета '%s' на строке %d: %v", itemName, lineNumber, err)
		}

		// Отладочное логирование для sell_consumables
		if currentCategory == "sell_consumables" {
			h.logger.Info("🔍 DEBUG: Вставляем предмет '%s' с категорией '%s' и ценой %.2f", itemName, currentCategory, minPrice)
		}

		// Подсчитываем количество предметов по категориям
		switch currentCategory {
		case "buy_consumables":
			buyConsumablesCount++
		case "buy_equipment":
			buyEquipmentCount++
		case "sell_consumables":
			sellConsumablesCount++
		case "sell_equipment":
			sellEquipmentCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}

	// Подтверждаем транзакцию
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка подтверждения транзакции: %v", err)
	}

	h.logger.Info("🔍 DEBUG: Транзакция подтверждена успешно")

	h.logger.Info("📊 Загружено предметов: %d buy_consumables, %d buy_equipment, %d sell_consumables, %d sell_equipment",
		buyConsumablesCount, buyEquipmentCount, sellConsumablesCount, sellEquipmentCount)

	return nil
}

// GetItemsList возвращает список всех предметов из базы данных
func (h *DatabaseManager) GetItemsList() ([]string, error) {
	rows, err := h.db.Query("SELECT name FROM items_list ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса предметов: %v", err)
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var itemName string
		err := rows.Scan(&itemName)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения предмета: %v", err)
		}
		items = append(items, itemName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по предметам: %v", err)
	}

	return items, nil
}

// GetItemsByCategory возвращает список предметов определенной категории
func (h *DatabaseManager) GetItemsByCategory(category string) ([]string, error) {
	rows, err := h.db.Query("SELECT name FROM items_list WHERE category = ? ORDER BY id", category)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса предметов категории %s: %v", category, err)
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("ошибка сканирования предмета: %v", err)
		}
		items = append(items, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по предметам: %v", err)
	}

	return items, nil
}

// GetItemsWithCategories возвращает список предметов с их категориями
func (h *DatabaseManager) GetItemsWithCategories() (map[string][]string, error) {
	rows, err := h.db.Query("SELECT name, category FROM items_list ORDER BY category, id")
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса предметов с категориями: %v", err)
	}
	defer rows.Close()

	itemsByCategory := make(map[string][]string)
	for rows.Next() {
		var itemName, category string
		err := rows.Scan(&itemName, &category)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения предмета: %v", err)
		}
		itemsByCategory[category] = append(itemsByCategory[category], itemName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по предметам: %v", err)
	}

	return itemsByCategory, nil
}

// GetItemIDByName возвращает ID предмета по названию
func (h *DatabaseManager) GetItemIDByName(itemName string) (int, error) {
	var id int
	err := h.db.QueryRow("SELECT id FROM items_list WHERE name = ?", itemName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения ID предмета '%s': %v", itemName, err)
	}
	return id, nil
}

// GetItemCategory возвращает категорию конкретного предмета
func (h *DatabaseManager) GetItemCategory(itemName string) (string, error) {
	var category string
	err := h.db.QueryRow("SELECT category FROM items_list WHERE name = ?", itemName).Scan(&category)
	if err != nil {
		return "", fmt.Errorf("ошибка получения категории предмета '%s': %v", itemName, err)
	}
	return category, nil
}

func (h *DatabaseManager) RecreateItemsTable() error {
	h.logger.Info("🔄 Пересоздаем таблицу items_list...")

	// Удаляем внешний ключ, если он есть
	_, err := h.db.Exec("ALTER TABLE structured_items DROP FOREIGN KEY structured_items_ibfk_2")
	if err != nil {
		h.logger.Info("⚠️ Не удалось удалить внешний ключ structured_items_ibfk_2: %v", err)
	}

	// Удаляем существующую таблицу
	_, err = h.db.Exec("DROP TABLE IF EXISTS items_list")
	if err != nil {
		return fmt.Errorf("ошибка удаления таблицы items_list: %v", err)
	}

	// Создаем таблицу заново
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS items_list (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		category VARCHAR(50) NOT NULL DEFAULT 'consumables',
		min_price DECIMAL(15,2) DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY unique_item_category (name, category)
	)`

	_, err = h.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы items_list: %v", err)
	}

	// Очищаем ссылки на items_list в structured_items
	_, err = h.db.Exec("UPDATE structured_items SET item_list_id = NULL")
	if err != nil {
		return fmt.Errorf("ошибка очистки ссылок в structured_items: %v", err)
	}

	// Восстанавливаем внешний ключ
	_, err = h.db.Exec(`
		ALTER TABLE structured_items
		ADD CONSTRAINT structured_items_ibfk_2
		FOREIGN KEY (item_list_id) REFERENCES items_list(id)
		ON DELETE SET NULL
	`)
	if err != nil {
		return fmt.Errorf("ошибка добавления внешнего ключа: %v", err)
	}

	h.logger.Info("✅ Таблица items_list пересоздана успешно")
	return nil
}

// GetCurrentStatus получает текущий статус из базы данных
func (h *DatabaseManager) GetCurrentStatus() (string, error) {
	var status string
	err := h.db.QueryRow("SELECT current_status FROM status ORDER BY created_at DESC LIMIT 1").Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "stopped", nil // По умолчанию считаем что система остановлена
		}
		return "", err
	}
	return status, nil
}

// GetLatestUnexecutedAction получает последнее невыполненное действие
func (h *DatabaseManager) GetLatestUnexecutedAction() (string, int, error) {
	var action string
	var id int
	err := h.db.QueryRow("SELECT id, action FROM actions WHERE executed = 0 ORDER BY created_at DESC LIMIT 1").Scan(&id, &action)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, nil // Нет невыполненных действий
		}
		return "", 0, err
	}
	return action, id, nil
}

// MarkActionAsExecuted помечает действие как выполненное
func (h *DatabaseManager) MarkActionAsExecuted(actionID int) error {
	_, err := h.db.Exec("UPDATE actions SET executed = 1 WHERE id = ?", actionID)
	return err
}

// UpdateStatus обновляет статус в базе данных
func (h *DatabaseManager) UpdateStatus(status string) error {
	_, err := h.db.Exec("INSERT INTO status (current_status) VALUES (?)", status)
	return err
}
