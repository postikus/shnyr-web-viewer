package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// SaveStructuredDataBatch сохраняет структурированные данные в базу данных
func SaveStructuredDataBatch(db *sql.DB, ocrResultID int, jsonData string) error {
	var items []map[string]any
	// Сначала пробуем как массив
	err := json.Unmarshal([]byte(jsonData), &items)
	if err != nil {
		// Если не получилось, пробуем как объект
		var single map[string]interface{}
		err2 := json.Unmarshal([]byte(jsonData), &single)
		if err2 != nil {
			return fmt.Errorf("ошибка парсинга JSON: %v", err)
		}
		items = append(items, single)
	}

	// Создаем таблицу для структурированных данных, если она не существует
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS structured_items (
		id INT AUTO_INCREMENT PRIMARY KEY,
		ocr_result_id INT NOT NULL,
		item_data JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ocr_result_id) REFERENCES ocr_results(id) ON DELETE CASCADE
	)`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы structured_items: %v", err)
	}

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Подготавливаем запрос для вставки
	insertSQL := `INSERT INTO structured_items (ocr_result_id, item_data) VALUES (?, ?)`
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса: %v", err)
	}
	defer stmt.Close()

	// Вставляем каждый элемент
	for _, item := range items {
		itemJSON, err := json.Marshal(item)
		if err != nil {
			log.Printf("⚠️ Ошибка сериализации элемента: %v", err)
			continue
		}

		_, err = stmt.Exec(ocrResultID, string(itemJSON))
		if err != nil {
			log.Printf("⚠️ Ошибка вставки элемента: %v", err)
			continue
		}
	}

	// Подтверждаем транзакцию
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка подтверждения транзакции: %v", err)
	}

	log.Printf("✅ Сохранено %d структурированных элементов для OCR ID: %d", len(items), ocrResultID)
	return nil
}
