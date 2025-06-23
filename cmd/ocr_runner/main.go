package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// 1. Проверяем, передан ли хотя бы один путь к файлу.
	if len(os.Args) < 2 {
		log.Fatalf("Пожалуйста, укажите один или несколько путей к файлам скриншотов. Пример: go run ./cmd/ocr_runner/main.go ./imgs/screenshot1.png ./imgs/screenshot2.png")
	}

	var screenshotFiles []string
	debugMode := false
	for _, arg := range os.Args[1:] {
		if arg == "debug=1" {
			debugMode = true
		} else {
			screenshotFiles = append(screenshotFiles, arg)
		}
	}

	if debugMode {
		fmt.Println("Debug mode enabled")
	}
	// 2. Путь к исполняемому файлу OCR.
	ocrExecutable := `C:\Users\karpo\cpp_ocr\build\Release\cpp_ocr.exe`

	fmt.Printf("Запускаю OCR для %d файлов...\n", len(screenshotFiles))

	// 3. Проходим по каждому переданному файлу и запускаем для него OCR.
	for _, fileArg := range screenshotFiles {
		// Формируем полный путь к файлу скриншота.
		screenshotPath, err := filepath.Abs(fileArg)
		if err != nil {
			log.Printf("Не удалось получить абсолютный путь для '%s': %v", fileArg, err)
			continue
		}

		fmt.Printf("\n--- Обработка файла: %s ---\n", fileArg)

		// Создаем команду для выполнения.
		cmd := exec.Command(ocrExecutable, screenshotPath)
		if debugMode {
			fmt.Printf("Executing command: %s\n", cmd.String())
		}
		// Выполняем команду и получаем ее вывод.
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Ошибка при выполнении OCR для '%s': %v", fileArg, err)
			log.Printf("Вывод команды: %s", string(output))
			continue
		}

		// Печатаем результат.
		fmt.Printf("Результат:\n%s\n", string(output))
	}

	fmt.Println("\nОбработка завершена.")
}
