package ocr

import (
	"fmt"
	"os/exec"
	"strings"
)

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

// ParseOCRResult парсит результат OCR и извлекает debug информацию и JSON
func ParseOCRResult(ocrResult string) (debugInfo, jsonData string) {
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
	} else {
		// Если маркеры не найдены, всё считаем debug информацией
		debugInfo = ocrResult
		jsonData = ""
	}

	return debugInfo, jsonData
}
