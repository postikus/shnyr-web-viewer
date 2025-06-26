package database

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"shnyr/internal/config"
	"shnyr/internal/logger"
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
func (h *DatabaseManager) SaveOCRResultToDB(imagePath, ocrResult string, debugInfo, jsonData string, rawText string, imageData []byte, cfg *config.Config) (int, error) {
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
			err := SaveStructuredDataBatch(h.db, ocrID, jsonStr)
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
func (h *DatabaseManager) InitializeItemsTable(filename string) error {
	// Создаем таблицу предметов с категориями
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS items_list (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		category VARCHAR(50) NOT NULL DEFAULT 'consumables',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := h.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы предметов: %v", err)
	}

	// Очищаем таблицу перед загрузкой новых данных
	h.logger.Info("🧹 Очищаем таблицу предметов")
	_, err = h.db.Exec("DELETE FROM items_list")
	if err != nil {
		return fmt.Errorf("ошибка очистки таблицы предметов: %v", err)
	}

	// Сбрасываем автоинкремент
	_, err = h.db.Exec("ALTER TABLE items_list AUTO_INCREMENT = 1")
	if err != nil {
		return fmt.Errorf("ошибка сброса автоинкремента: %v", err)
	}

	// Загружаем данные из файла
	h.logger.Info("📁 Загружаем предметы из файла: %s", filename)
	err = h.loadItemsFromFile(filename)
	if err != nil {
		return fmt.Errorf("ошибка загрузки предметов из файла: %v", err)
	}
	h.logger.Info("✅ Предметы успешно загружены из файла")

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
	stmt, err := tx.Prepare("INSERT IGNORE INTO items_list (name, category) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса: %v", err)
	}
	defer stmt.Close()

	lineNumber := 0
	currentCategory := "buy_consumables" // По умолчанию первая категория - скупка расходников
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
				currentCategory = "sell_consumables"
			case "sell_consumables":
				currentCategory = "sell_equipment"
			default:
				// Если это первый разделитель, переходим к buy_equipment
				currentCategory = "buy_equipment"
			}
			h.logger.Info("📋 Переключаемся на категорию: %s", currentCategory)
			continue
		case "===":
			currentCategory = "sell_consumables"
			h.logger.Info("📋 Переключаемся на категорию: %s", currentCategory)
			continue
		}

		// Вставляем предмет с категорией
		_, err := stmt.Exec(line, currentCategory)
		if err != nil {
			return fmt.Errorf("ошибка вставки предмета '%s' на строке %d: %v", line, lineNumber, err)
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

// GetItemsByCategory возвращает список предметов по категории
func (h *DatabaseManager) GetItemsByCategory(category string) ([]string, error) {
	rows, err := h.db.Query("SELECT name FROM items_list WHERE category = ? ORDER BY id", category)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса предметов категории %s: %v", category, err)
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

// GetItemCategory возвращает категорию конкретного предмета
func (h *DatabaseManager) GetItemCategory(itemName string) (string, error) {
	var category string
	err := h.db.QueryRow("SELECT category FROM items_list WHERE name = ?", itemName).Scan(&category)
	if err != nil {
		return "", fmt.Errorf("ошибка получения категории предмета '%s': %v", itemName, err)
	}
	return category, nil
}
