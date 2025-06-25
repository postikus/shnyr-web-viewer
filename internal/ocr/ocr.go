package ocr

import (
	"encoding/json"
	"fmt"
	"image"
	"shnyr/internal/config"
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

// OCRManager содержит функции для работы с OCR
type OCRManager struct {
	config *config.Config
}

// NewOCRManager создает новый экземпляр OCRManager
func NewOCRManager(config *config.Config) *OCRManager {
	return &OCRManager{
		config: config,
	}
}

// RunOCR запускает внешний cpp_ocr.exe и возвращает распознанный текст
func (m *OCRManager) RunOCR(imagePath string) (string, error) {
	ocrExecutable := `C:\Users\karpo\cpp_ocr\build\Release\cpp_ocr.exe`
	cmd := exec.Command(ocrExecutable, imagePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ошибка при выполнении OCR: %v, вывод: %s", err, string(output))
	}
	return string(output), nil
}

// fixMalformedJSON исправляет JSON с отсутствующими запятыми в массиве structured_data
func (m *OCRManager) fixMalformedJSON(jsonData string) string {
	// Ищем паттерн: } { в массиве structured_data
	// Это означает отсутствующую запятую между объектами
	pattern := regexp.MustCompile(`(\s*}\s*)(\s*{\s*"title":)`)
	replacement := `$1,$2`

	// Применяем замену
	fixedJSON := pattern.ReplaceAllString(jsonData, replacement)

	return fixedJSON
}

// ParseOCRResult парсит результат OCR и извлекает debug информацию, JSON и raw_text
func (m *OCRManager) ParseOCRResult(ocrResult string) (debugInfo, jsonData, rawText string) {
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
		jsonData = m.fixMalformedJSON(jsonData)

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
				jsonData = m.fixMalformedJSON(jsonData)

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

// ProcessImage выполняет OCR обработку изображения
func (m *OCRManager) ProcessImage(img image.Image, fileName string) (result, debugInfo, jsonData, rawText string, err error) {
	// Выполняем OCR
	result, err = m.RunOCR(fileName)
	if err != nil {
		fmt.Printf("Ошибка при выполнении OCR: %v\n", err)
		return "", "", "", "", err
	}

	// Парсим результат OCR
	debugInfo, jsonData, rawText = m.ParseOCRResult(result)

	return result, debugInfo, jsonData, rawText, nil
}
