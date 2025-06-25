package database

import (
	"database/sql"
	"fmt"
	"shnyr/internal/config"
	"shnyr/internal/logger"
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
