package ocr

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// StructuredItem представляет один элемент из structured_data
type StructuredItem struct {
	Title       string `json:"title"`
	TitleShort  string `json:"title_short"`
	Enhancement string `json:"enhancement"`
	Price       string `json:"price"`
	Package     bool   `json:"package"`
	Owner       string `json:"owner"`
	Count       string `json:"count"`
}

// OCRJSONResult представляет структуру JSON ответа
type OCRJSONResult struct {
	ImageFile  string `json:"image_file"`
	Processing struct {
		Enlargement         string `json:"enlargement"`
		Grayscale           bool   `json:"grayscale"`
		Denoising           string `json:"denoising"`
		ContrastEnhancement string `json:"contrast_enhancement"`
		Binarization        string `json:"binarization"`
		OCREngine           string `json:"ocr_engine"`
		OCRLanguages        string `json:"ocr_languages"`
		OCRMode             string `json:"ocr_mode"`
	} `json:"processing"`
	TextRecognition struct {
		Success        bool             `json:"success"`
		RawText        string           `json:"raw_text"`
		StructuredData []StructuredItem `json:"structured_data"`
		Confidence     string           `json:"confidence"`
	} `json:"text_recognition"`
}

// RunOCR запускает внешний cpp_ocr.exe и возвращает распознанный текст
func RunOCR(imagePath string) (string, error) {
	ocrExecutable := `C:\Users\karpo\cpp_ocr\build\Release\cpp_ocr.exe`
	cmd := exec.Command(ocrExecutable, imagePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ошибка при выполнении OCR: %v, вывод: %s", err, string(output))
	}
	return string(output), nil
}

// fixMalformedJSON исправляет JSON с отсутствующими запятыми в массиве structured_data
func fixMalformedJSON(jsonData string) string {
	// Ищем паттерн: } { в массиве structured_data
	// Это означает отсутствующую запятую между объектами
	pattern := regexp.MustCompile(`(\s*}\s*)(\s*{\s*"title":)`)
	replacement := `$1,$2`

	// Применяем замену
	fixedJSON := pattern.ReplaceAllString(jsonData, replacement)

	return fixedJSON
}

// ParseOCRResult парсит результат OCR и извлекает debug информацию, JSON и raw_text
func ParseOCRResult(ocrResult string) (debugInfo, jsonData, rawText string) {
	// Ищем маркеры JSON
	jsonStart := "=== JSON START ==="
	jsonEnd := "=== JSON END ==="

	startIndex := strings.Index(ocrResult, jsonStart)
	endIndex := strings.Index(ocrResult, jsonEnd)

	if startIndex != -1 && endIndex != -1 && endIndex > startIndex {
		// Извлекаем debug информацию (всё до JSON)
		debugInfo = strings.TrimSpace(ocrResult[:startIndex])

		// Извлекаем JSON (между маркерами)
		jsonStartPos := startIndex + len(jsonStart)
		jsonData = strings.TrimSpace(ocrResult[jsonStartPos:endIndex])

		// Исправляем malformed JSON
		jsonData = fixMalformedJSON(jsonData)

		// Извлекаем raw_text из JSON
		var ocrResult OCRJSONResult
		if err := json.Unmarshal([]byte(jsonData), &ocrResult); err == nil {
			rawText = ocrResult.TextRecognition.RawText
		}
	} else {
		// Если маркеры не найдены, пытаемся найти JSON в выводе
		// Ищем начало JSON (первая открывающая скобка)
		jsonStartPos := strings.Index(ocrResult, "{")
		if jsonStartPos != -1 {
			// Ищем конец JSON (последняя закрывающая скобка)
			jsonEndPos := strings.LastIndex(ocrResult, "}")
			if jsonEndPos != -1 && jsonEndPos > jsonStartPos {
				// Извлекаем debug информацию (всё до JSON)
				debugInfo = strings.TrimSpace(ocrResult[:jsonStartPos])

				// Извлекаем JSON
				jsonData = strings.TrimSpace(ocrResult[jsonStartPos : jsonEndPos+1])

				// Исправляем malformed JSON
				jsonData = fixMalformedJSON(jsonData)

				// Извлекаем raw_text из JSON
				var ocrResult OCRJSONResult
				if err := json.Unmarshal([]byte(jsonData), &ocrResult); err == nil {
					rawText = ocrResult.TextRecognition.RawText
				} else {
					fmt.Printf("Ошибка парсинга JSON без маркеров: %v\n", err)
					fmt.Printf("JSON данные: %s\n", jsonData)
				}
			} else {
				// Если не можем найти JSON, всё считаем debug информацией
				debugInfo = ocrResult
				jsonData = ""
				rawText = ""
			}
		} else {
			// Если не можем найти JSON, всё считаем debug информацией
			debugInfo = ocrResult
			jsonData = ""
			rawText = ""
		}
	}

	return debugInfo, jsonData, rawText
}

// SaveStructuredData сохраняет структурированные данные в базу данных
func SaveStructuredData(db *sql.DB, ocrResultID int, jsonData string) error {
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
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ocr_result_id) REFERENCES ocr_results(id) ON DELETE CASCADE
	)`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы structured_items: %v", err)
	}

	// Если нет данных для сохранения, выходим
	if len(ocrResult.TextRecognition.StructuredData) == 0 {
		fmt.Printf("Нет структурированных данных для сохранения (OCR ID: %d)\n", ocrResultID)
		return nil
	}

	// Начинаем транзакцию для batch операций
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
	stmt, err := tx.Prepare(`INSERT INTO structured_items (ocr_result_id, title, title_short, enhancement, price, package, owner, count) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса: %v", err)
	}
	defer stmt.Close()

	// Выполняем batch вставку
	for _, item := range ocrResult.TextRecognition.StructuredData {
		// Устанавливаем "0" для пустого enhancement
		enhancement := item.Enhancement
		if enhancement == "" {
			enhancement = "0"
			fmt.Printf("🔧 Установлен enhancement='0' для предмета: %s\n", item.Title)
		}

		_, err = stmt.Exec(ocrResultID, item.Title, item.TitleShort, enhancement, item.Price, item.Package, item.Owner, item.Count)
		if err != nil {
			return fmt.Errorf("ошибка вставки структурированных данных: %v", err)
		}
	}

	// Фиксируем транзакцию
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка фиксации транзакции: %v", err)
	}

	fmt.Printf("✅ Сохранено %d структурированных элементов для OCR результата ID: %d (batch операция)\n",
		len(ocrResult.TextRecognition.StructuredData), ocrResultID)
	return nil
}

// SaveStructuredDataBatch сохраняет структурированные данные в базу данных используя один INSERT запрос с множественными VALUES
func SaveStructuredDataBatch(db *sql.DB, ocrResultID int, jsonData string) error {
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
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ocr_result_id) REFERENCES ocr_results(id) ON DELETE CASCADE
	)`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы structured_items: %v", err)
	}

	// Если нет данных для сохранения, выходим
	if len(ocrResult.TextRecognition.StructuredData) == 0 {
		fmt.Printf("Нет структурированных данных для сохранения (OCR ID: %d)\n", ocrResultID)
		return nil
	}

	// Строим один INSERT запрос с множественными VALUES
	var values []string
	var args []interface{}

	for _, item := range ocrResult.TextRecognition.StructuredData {
		// Устанавливаем "0" для пустого enhancement
		enhancement := item.Enhancement
		if enhancement == "" {
			enhancement = "0"
			fmt.Printf("🔧 Установлен enhancement='0' для предмета: %s\n", item.Title)
		}

		values = append(values, "(?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args, ocrResultID, item.Title, item.TitleShort, enhancement, item.Price, item.Package, item.Owner, item.Count)
	}

	// Формируем SQL запрос
	insertSQL := fmt.Sprintf(`INSERT INTO structured_items (ocr_result_id, title, title_short, enhancement, price, package, owner, count) VALUES %s`, strings.Join(values, ","))

	// Выполняем один batch запрос
	_, err = db.Exec(insertSQL, args...)
	if err != nil {
		return fmt.Errorf("ошибка batch вставки структурированных данных: %v", err)
	}

	fmt.Printf("🚀 Сохранено %d структурированных элементов для OCR результата ID: %d (один INSERT запрос)\n",
		len(ocrResult.TextRecognition.StructuredData), ocrResultID)
	return nil
}
