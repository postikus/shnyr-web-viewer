package ocr

import (
	"fmt"
	"os/exec"
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
