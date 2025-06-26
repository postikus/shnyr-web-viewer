package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// SaveStructuredDataBatch сохраняет структурированные данные в базу данных
func SaveStructuredDataBatch(db *sql.DB, ocrResultID int, jsonData string, itemCategory string, currentItemName string) error {
	if jsonData == "" {
		return nil // Нет данных для сохранения
	}

	// Парсим JSON
	var ocrResult OCRJSONResult
	err := json.Unmarshal([]byte(jsonData), &ocrResult)
	if err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	// Создаем таблицу, если она не существует
	createTableSQL := `CREATE TABLE IF NOT EXISTS structured_items (
		id INT AUTO_INCREMENT PRIMARY KEY,
		ocr_result_id INT,
		title VARCHAR(255) NOT NULL,
		title_short VARCHAR(255),
		enhancement VARCHAR(10),
		price VARCHAR(50) NOT NULL,
		package BOOLEAN DEFAULT FALSE,
		owner VARCHAR(255),
		count VARCHAR(10),
		category VARCHAR(50),
		item_list_id INT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ocr_result_id) REFERENCES ocr_results(id) ON DELETE CASCADE,
		FOREIGN KEY (item_list_id) REFERENCES items_list(id) ON DELETE SET NULL
	)`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы structured_items: %v", err)
	}

	// Если нет данных для сохранения, выходим
	if len(ocrResult.TextRecognition.StructuredData) == 0 {
		fmt.Printf("ℹ️ Нет структурированных данных для сохранения (OCR ID: %d)\n", ocrResultID)
		return nil
	}

	// Получаем ID текущего предмета из items_list
	var itemListID *int
	if currentItemName != "" {
		var id int
		err := db.QueryRow("SELECT id FROM items_list WHERE name = ?", currentItemName).Scan(&id)
		if err == nil {
			itemListID = &id
		} else {
			fmt.Printf("⚠️ Не удалось найти предмет '%s' в items_list: %v\n", currentItemName, err)
		}
	}

	// Начинаем транзакцию для batch обработки
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Подготавливаем запрос для batch вставки
	insertSQL := `INSERT INTO structured_items (ocr_result_id, title, title_short, enhancement, price, package, owner, count, category, item_list_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса: %v", err)
	}
	defer stmt.Close()

	// Сохраняем каждый элемент в batch
	processedCount := 0
	for _, item := range ocrResult.TextRecognition.StructuredData {
		// Устанавливаем "0" для пустого enhancement
		enhancement := item.Enhancement
		if enhancement == "" {
			enhancement = "0"
		}

		_, err = stmt.Exec(ocrResultID, item.Title, item.TitleShort, enhancement, item.Price, item.Package, item.Owner, item.Count, itemCategory, itemListID)
		if err != nil {
			return fmt.Errorf("ошибка вставки структурированных данных: %v", err)
		}
		processedCount++
	}

	// Подтверждаем транзакцию
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка подтверждения транзакции: %v", err)
	}

	fmt.Printf("✅ Сохранено %d/%d структурированных элементов для OCR результата ID: %d (категория: %s, item_list_id: %v)\n",
		processedCount, len(ocrResult.TextRecognition.StructuredData), ocrResultID, itemCategory, itemListID)
	return nil
}
