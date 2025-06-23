package scpript1

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"octopus/internal/config"
	imageInternal "octopus/internal/image"
	"octopus/internal/ocr"
	"octopus/internal/screen"
	"octopus/internal/screenshot"
	"octopus/internal/scripts"
	"os"

	"github.com/tarm/serial"
)

var Run = func(port *serial.Port, c *config.Config, db *sql.DB) {

	// Количество пикселей для отрезания сверху
	topOffset := 23

	// Делаем скриншот всего экрана
	img, err := screen.CaptureFullScreen()
	if err != nil {
		log.Fatalf("Ошибка захвата экрана: %v", err)
	}

	// Обрезаем верхние пиксели
	bounds := img.Bounds()
	croppedImg := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()-topOffset))
	for y := topOffset; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			croppedImg.Set(x, y-topOffset, img.At(x, y))
		}
	}

	// Ищем окно
	gameWindow, err := imageInternal.FindGameWindow(croppedImg)
	if err != nil {
		fmt.Printf("Окно не найдено: %v\n", err)
		return
	}

	marginX := gameWindow.X - 150
	marginY := gameWindow.Y + topOffset + 48

	fmt.Printf("marginX, marginY: %v %v\n", marginX, marginY)

	var captureScreenShot = func() image.Image {
		img, _ := screenshot.CaptureScreenshot(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 361})
		return img
	}

	var saveScreenShot = func() image.Image {
		img, _ := screenshot.SaveScreenshot(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 361}, c)
		return img
	}

	var saveScreenShotFull = func() image.Image {
		img, _ := screenshot.SaveScreenshotFull(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 361})
		return img
	}

	var checkAndScreenScroll = func(counter int, x int) (int, int) {
		img := captureScreenShot()
		r, _, _, _ := imageInternal.GetPixelColor(img, 297, 320)
		fmt.Printf("r: %v\n", r)
		if r < 50 {
			scripts.ScrollDown(port, c, x)
		}
		return counter + 1, r
	}

	var checkAndClickScreenScroll = func(counter int) (int, int) {
		img := captureScreenShot()
		r, _, _, _ := imageInternal.GetPixelColor(img, 297, 342)
		if r < 50 {
			scripts.FastClick(port, c)
		}
		return counter + 1, r
	}

	countFilesInDir := func(dir string) (int, error) {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return 0, fmt.Errorf("не удалось прочитать папку: %v", err)
		}

		// Возвращаем количество файлов в папке
		return len(files), nil
	}

	// Функция для проверки активности кнопки
	var checkButtonActive = func(buttonX, buttonY int, buttonName string) bool {
		img := captureScreenShot()
		buttonRPx, _, _, _ := imageInternal.GetPixelColor(img, buttonX, 36)
		fmt.Printf("%s RPx: %v\n", buttonName, buttonRPx)
		return buttonRPx == 86
	}

	// Функция для выполнения полного цикла скриншотов и OCR
	var performScreenshotAndOCR = func(buttonPressed bool) error {
		counter := 0
		maxCounter := 40
		scrollRPx := 26

		// Список для хранения всех скриншотов
		var screenshots []image.Image
		var smallScreenshots []image.Image

		img := captureScreenShot()
		saveScreenShotFull()
		screenshots = append(screenshots, img)

		scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
		fmt.Printf("scrollRPx: %v %v %v\n", scrollRPx, scrollGPx, scrollBPx)

		// Проверяем наличие кнопок 1, 2 или 3
		button1Active := checkButtonActive(c.Click.Button1.X, c.Click.Button1.Y, "listButton1")
		button2Active := checkButtonActive(c.Click.Button2.X, c.Click.Button2.Y, "listButton2")
		button3Active := checkButtonActive(c.Click.Button3.X, c.Click.Button3.Y, "listButton3")

		topCrop := 22 // По умолчанию обрезаем 22 пикселя сверху
		if buttonPressed || button1Active || button2Active || button3Active {
			topCrop = 45
		}

		if scrollRPx > 26 {
			scrollRPx = 26
			for counter < maxCounter && scrollRPx < 50 {
				counter, scrollRPx = checkAndScreenScroll(counter, 1)
				if scrollRPx < 50 {
					img = captureScreenShot()
					screenshots = append(screenshots, img)
				}
			}

			scrollRPx = 26
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Scroll.X, Y: marginY + c.Click.Scroll.Y})
			for counter < maxCounter && scrollRPx < 50 {
				counter, scrollRPx = checkAndClickScreenScroll(counter)
				if scrollRPx < 50 {
					img = captureScreenShot()
					smallScreenshots = append(smallScreenshots, img)
				}
			}

			var finalImage image.Image
			// Проверка stripe_diff между последним и предпоследним smallScreenshots
			if len(smallScreenshots) >= 2 {
				prev := smallScreenshots[len(smallScreenshots)-2]
				last := smallScreenshots[len(smallScreenshots)-1]
				// Сохраняем два последних скриншота в файлы
				f1, _ := os.Create("last_prev.png")
				defer f1.Close()
				png.Encode(f1, prev)
				f2, _ := os.Create("last.png")
				defer f2.Close()
				png.Encode(f2, last)
				diff, err := imageInternal.LastColorStripeDistanceDiff(prev, last, 26, 20)
				if err != nil {
					fmt.Printf("Ошибка stripe diff: %v\n", err)
				} else {
					finalImage, _ = imageInternal.CombineImages(screenshots, smallScreenshots[:len(smallScreenshots)-1], smallScreenshots[len(smallScreenshots)-1], diff)
					fmt.Printf("Разница stripe diff между последним и предпоследним скриншотом: %d\n", diff)
				}
			} else if len(smallScreenshots) == 1 {
				prev := screenshots[len(screenshots)-1]
				last := smallScreenshots[len(smallScreenshots)-1]
				diff, err := imageInternal.LastColorStripeDistanceDiff(prev, last, 26, 20)
				if err != nil {
					fmt.Printf("Ошибка stripe diff: %v\n", err)
				} else {
					finalImage, _ = imageInternal.CombineImages(screenshots, smallScreenshots[:len(smallScreenshots)-1], nil, 0)
					fmt.Printf("Разница stripe diff между последним и предпоследним скриншотом: %d\n", diff)
				}
			} else {
				finalImage, _ = imageInternal.CombineImages(screenshots, nil, nil, 0)
			}

			combinedImg := imageInternal.CropOpacityPixel(finalImage)
			bounds := combinedImg.Bounds()
			cropRect := image.Rect(40, topCrop, bounds.Dx()-17, bounds.Dy())
			croppedCombinedImg := combinedImg.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(cropRect)
			img = croppedCombinedImg
		} else {
			// Если скролла нет, просто кадрируем первый скриншот
			fmt.Println("⚠️ scrollRPx <= 26, делаем обычный скриншот с кадрированием")
			bounds := img.Bounds()
			cropRect := image.Rect(40, topCrop, bounds.Dx()-17, bounds.Dy())
			croppedCombinedImg := img.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(cropRect)
			img = croppedCombinedImg
		}

		fileCount, _ := countFilesInDir("./imgs")
		fileName := fmt.Sprintf("%s/screenshot_combined_%d.png", "./imgs", fileCount)
		err := imageInternal.SaveCombinedImage(img, fileName)
		if err != nil {
			return err
		}

		scripts.ScrollUp(port, c, counter+5)

		result, err := ocr.RunOCR(fileName)
		if err != nil {
			fmt.Printf("Ошибка при выполнении OCR: %v\n", err)
			return err
		}

		fmt.Println(result)

		// Парсим результат OCR
		debugInfo, jsonData, rawText := ocr.ParseOCRResult(result)

		// Конвертируем изображение в байты
		imageBytes, err := imageToBytes(img)
		if err != nil {
			log.Printf("Ошибка конвертации изображения: %v", err)
			return err
		}

		// Сохраняем результат в базу данных
		_, err = saveOCRResultToDB(db, fileName, result, debugInfo, jsonData, rawText, imageBytes, c)
		if err != nil {
			log.Printf("Ошибка сохранения в БД: %v", err)
			return err
		}

		return nil
	}

	var captureScreenShotsWithScroll = func() bool {
		fmt.Println("=== Начало выполнения captureScreenShotsWithScroll ===")

		// Сначала проверяем, есть ли скролл вообще
		img := captureScreenShot()
		scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
		fmt.Printf("scrollRPx: %v %v %v\n", scrollRPx, scrollGPx, scrollBPx)

		// Если нет скролла, возвращаем false
		if scrollRPx <= 26 {
			fmt.Println("❌ Скролл не найден (scrollRPx <= 26), выходим из функции")
			return false
		}
		fmt.Println("✅ Скролл найден, продолжаем выполнение")

		// Проверяем наличие всех кнопок
		fmt.Println("🔍 Проверяем наличие кнопок...")
		// SaveScreenshotFull()
		button2Active := checkButtonActive(c.Click.Button2.X, c.Click.Button2.Y, "listButton2")
		button3Active := checkButtonActive(c.Click.Button3.X, c.Click.Button3.Y, "listButton3")
		button4Active := checkButtonActive(c.Click.Button4.X, c.Click.Button4.Y, "listButton4")
		button5Active := checkButtonActive(c.Click.Button5.X, c.Click.Button5.Y, "listButton5")
		button6Active := checkButtonActive(c.Click.Button6.X, c.Click.Button6.Y, "listButton6")

		fmt.Printf("📋 Статус кнопок: Button2=%v, Button3=%v, Button4=%v, Button5=%v, Button6=%v\n",
			button2Active, button3Active, button4Active, button5Active, button6Active)

		// Выполняем основной цикл скриншотов и OCR (без нажатия кнопок)
		fmt.Println("🔄 Выполняем основной цикл скриншотов и OCR...")
		if button2Active {
			err := performScreenshotAndOCR(true)
			if err != nil {
				fmt.Printf("❌ Ошибка в основном цикле: %v\n", err)
				return false
			}
		} else {
			err := performScreenshotAndOCR(false)
			if err != nil {
				fmt.Printf("❌ Ошибка в основном цикле: %v\n", err)
				return false
			}
		}

		fmt.Println("✅ Основной цикл выполнен успешно")

		// Идем по кнопкам последовательно
		if button2Active {
			fmt.Println("🔘 Кликаем по Button2...")
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button2.X, Y: marginY + c.Click.Button2.Y})
			err := performScreenshotAndOCR(true)
			if err != nil {
				fmt.Printf("❌ Ошибка при обработке Button2: %v\n", err)
				return false
			}
			fmt.Println("✅ Button2 обработан успешно")
		} else {
			fmt.Println("⏭️ Button2 неактивен, пропускаем")
		}

		if button3Active {
			fmt.Println("🔘 Кликаем по Button3...")
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button3.X, Y: marginY + c.Click.Button3.Y})
			err := performScreenshotAndOCR(true)
			if err != nil {
				fmt.Printf("❌ Ошибка при обработке Button3: %v\n", err)
				return false
			}
			fmt.Println("✅ Button3 обработан успешно")
		} else {
			fmt.Println("⏭️ Button3 неактивен, пропускаем")
		}

		if button4Active {
			fmt.Println("🔘 Кликаем по Button4...")
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button4.X, Y: marginY + c.Click.Button4.Y})
			err := performScreenshotAndOCR(true)
			if err != nil {
				fmt.Printf("❌ Ошибка при обработке Button4: %v\n", err)
				return false
			}
			fmt.Println("✅ Button4 обработан успешно")
		} else {
			fmt.Println("⏭️ Button4 неактивен, пропускаем")
		}

		if button5Active {
			fmt.Println("🔘 Кликаем по Button5...")
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button5.X, Y: marginY + c.Click.Button5.Y})
			err := performScreenshotAndOCR(true)
			if err != nil {
				fmt.Printf("❌ Ошибка при обработке Button5: %v\n", err)
				return false
			}
			fmt.Println("✅ Button5 обработан успешно")
		} else {
			fmt.Println("⏭️ Button5 неактивен, пропускаем")
		}

		if button6Active {
			fmt.Println("🔘 Кликаем по Button6...")
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button6.X, Y: marginY + c.Click.Button6.Y})
			err := performScreenshotAndOCR(true)
			if err != nil {
				fmt.Printf("❌ Ошибка при обработке Button6: %v\n", err)
				return false
			}
			fmt.Println("✅ Button6 обработан успешно")
		} else {
			fmt.Println("⏭️ Button6 неактивен, пропускаем")
		}

		// Кликаем Back только после последней существующей кнопки
		fmt.Println("🔙 Кликаем по кнопке Back...")
		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
		fmt.Println("✅ Back клик выполнен")

		fmt.Println("=== Завершение captureScreenShotsWithScroll ===")
		return true
	}

	var clickItem = func(item config.Coordinates) {
		scripts.ClickCoordinates(port, c, item)
		combinedSaved := captureScreenShotsWithScroll()
		if !combinedSaved {
			saveScreenShot()
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
		}

	}

	var clickEveryItemAnsScreenShot = func(img image.Image) {
		// прокликиваем первую страницу
		points := imageInternal.FindItemPositionsByTextColor(img, 80)
		fmt.Printf("🔍 Найдено точек для клика: %d\n", len(points))
		if len(points) > 0 {
			fmt.Printf("✅ Найдено достаточно точек, начинаем обработку...\n")
			for i, point := range points {
				fmt.Printf("🖱️ Кликаем по точке %d: (%d, %d)\n", i+1, point.X, point.Y)
				clickItem(config.Coordinates{Y: point.Y + marginY, X: marginX + point.X})
			}
		} else {
			fmt.Printf("⚠️ Недостаточно точек для обработки (нужно > 0, найдено: %d)\n", len(points))
		}

		// clickItem(config.Coordinates{X: marginX + 80, Y: marginY + c.Click.Item1.Y})
	}

	// // берем в фокус и делаем скрин
	// scripts.ClickCoordinates(port, c, c.Click.Item1)
	// img = captureScreenShot()
	// clickEveryItemAnsScreenShot(img)

	// берем в фокус
	scripts.ClickCoordinates(port, c, c.Click.Item1)

	cycles := 0
	for cycles < 2 {
		img := captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button2.X, Y: marginY + c.Click.Button2.Y})
		img = captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button3.X, Y: marginY + c.Click.Button3.Y})
		img = captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button4.X, Y: marginY + c.Click.Button4.Y})
		img = captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button5.X, Y: marginY + c.Click.Button5.Y})
		img = captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button6.X, Y: marginY + c.Click.Button6.Y})
		img = captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		img = captureScreenShot()
		SixButtonPx, _, _, _ := imageInternal.GetPixelColor(img, c.Click.Button6.X, 35)
		maxSixButtonClicks := 0

		for SixButtonPx > 30 && maxSixButtonClicks < 50 {
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button6.X, Y: marginY + c.Click.Button6.Y})
			img = captureScreenShot()
			clickEveryItemAnsScreenShot(img)
			img = captureScreenShot()
			SixButtonPx, _, _, _ = imageInternal.GetPixelColor(img, c.Click.Button6.X, 35)
			maxSixButtonClicks += 1
		}

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
		// scripts.ClickCoordinates(port, c, config.Coordinates{X: 35, Y: 107})

		cycles += 1
	}

}

// saveOCRResultToDB сохраняет результат OCR в базу данных
func saveOCRResultToDB(db *sql.DB, imagePath, ocrResult string, debugInfo, jsonData string, rawText string, imageData []byte, cfg *config.Config) (int, error) {
	// Проверяем настройку сохранения в БД
	if cfg.SaveToDB != 1 {
		log.Printf("Сохранение в БД отключено (save_to_db = %d)", cfg.SaveToDB)
		return 0, nil
	}

	log.Printf("💾 Начинаем сохранение OCR результата в БД...")
	log.Printf("📄 JSON данные (длина: %d): %s", len(jsonData), jsonData)
	log.Printf("🔍 Debug info (длина: %d): %s", len(debugInfo), debugInfo[:min(100, len(debugInfo))])
	log.Printf("📝 Raw text (длина: %d): %s", len(rawText), rawText[:min(100, len(rawText))])

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

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания таблицы: %v", err)
	}

	// Вставляем результат OCR с изображением
	insertSQL := `INSERT INTO ocr_results (image_path, image_data, ocr_text, debug_info, json_data, raw_text) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := db.Exec(insertSQL, imagePath, imageData, ocrResult, debugInfo, jsonData, rawText)
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
		err = ocr.SaveStructuredData(db, int(ocrResultID), jsonData)
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

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// imageToBytes конвертирует изображение в байты в формате PNG
func imageToBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("ошибка кодирования изображения: %v", err)
	}
	return buf.Bytes(), nil
}
