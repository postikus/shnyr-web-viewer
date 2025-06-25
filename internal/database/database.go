package database

import (
	"database/sql"
	"fmt"
	"log"
	"octopus/internal/config"
	"octopus/internal/helpers"
)

// DatabaseManager содержит функции для работы с базой данных
type DatabaseManager struct {
	db *sql.DB
}

// NewDatabaseManager создает новый экземпляр DatabaseManager
func NewDatabaseManager(db *sql.DB) *DatabaseManager {
	return &DatabaseManager{
		db: db,
	}
}

// SaveOCRResultToDB сохраняет результат OCR в базу данных
func (h *DatabaseManager) SaveOCRResultToDB(imagePath, ocrResult string, debugInfo, jsonData string, rawText string, imageData []byte, cfg *config.Config) (int, error) {
	// Проверяем настройку сохранения в БД
	if cfg.SaveToDB != 1 {
		log.Printf("Сохранение в БД отключено (save_to_db = %d)", cfg.SaveToDB)
		return 0, nil
	}

	log.Printf("💾 Начинаем сохранение OCR результата в БД...")
	log.Printf("📄 JSON данные (длина: %d): %s", len(jsonData), jsonData)
	log.Printf("🔍 Debug info (длина: %d): %s", len(debugInfo), debugInfo[:helpers.Min(100, len(debugInfo))])
	log.Printf("📝 Raw text (длина: %d): %s", len(rawText), rawText[:helpers.Min(100, len(rawText))])

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

	log.Printf("✅ OCR результат сохранен с ID: %d", ocrResultID)

	// Сохраняем структурированные данные
	if jsonData != "" {
		log.Printf("🔧 Сохраняем структурированные данные для OCR ID: %d", ocrResultID)
		err = SaveStructuredDataBatch(h.db, int(ocrResultID), jsonData)
		if err != nil {
			log.Printf("❌ Ошибка сохранения структурированных данных: %v", err)
		} else {
			log.Printf("✅ Структурированные данные успешно сохранены")
		}
	} else {
		log.Printf("⚠️ JSON данные пустые, пропускаем сохранение structured items")
	}

	log.Printf("OCR результат и изображение сохранены в базу данных для файла: %s (ID: %d)", imagePath, ocrResultID)
	return int(ocrResultID), nil
}
