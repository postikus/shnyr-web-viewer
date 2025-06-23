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

	var _ = func() image.Image {
		img, _ := screenshot.SaveScreenshotFull(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 361})
		return img
	}

	var checkAndScreenScroll = func(counter int, x int) (int, int) {
		img := captureScreenShot()
		r, _, _, _ := imageInternal.GetPixelColor(img, 297, 315)
		fmt.Printf("r: %v\n", r)
		if r < 50 {
			scripts.ScrollDown(port, c, x)
		}
		return counter + 1, r
	}

	var checkAndClickScreenScroll = func(counter int) (int, int) {
		img := captureScreenShot()
		r, _, _, _ := imageInternal.GetPixelColor(img, 297, 332)
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

	var captureScreenShotsWithScroll = func() bool {
		counter := 0
		maxCounter := 20
		scrollRPx := 26

		// Список для хранения всех скриншотов
		var screenshots []image.Image
		var smallScreenshots []image.Image

		img := captureScreenShot()
		// saveScreenShotFull()
		screenshots = append(screenshots, img)
		scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
		fmt.Printf("scrollRPx: %v %v %v\n", scrollRPx, scrollGPx, scrollBPx)

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

			finalImage, _ := imageInternal.CombineImages(screenshots, smallScreenshots)
			combinedImg := imageInternal.CropOpacityPixel(finalImage)

			// --- Новая логика кадрирования для комбинированного изображения ---
			bounds := combinedImg.Bounds()
			cropRect := image.Rect(40, 22, bounds.Dx()-17, bounds.Dy())
			croppedCombinedImg := combinedImg.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(cropRect)
			// --- Конец логики кадрирования ---

			fileCount, _ := countFilesInDir("./imgs")
			fileName := fmt.Sprintf("%s/screenshot_combined_%d.png", "./imgs", fileCount)
			err = imageInternal.SaveCombinedImage(croppedCombinedImg, fileName)
			if err != nil {
				return false
			}

			scripts.ScrollUp(port, c, counter+5)

			result, err := ocr.RunOCR(fileName)

			if err != nil {
				fmt.Printf("Ошибка при выполнении OCR: %v\n", err)
			} else {
				fmt.Println(result)

				// Парсим результат OCR
				debugInfo, jsonData, rawText := ocr.ParseOCRResult(result)

				// Конвертируем изображение в байты
				imageBytes, err := imageToBytes(croppedCombinedImg)
				if err != nil {
					log.Printf("Ошибка конвертации изображения: %v", err)
				} else {
					// Сохраняем результат в базу данных
					_, err = saveOCRResultToDB(db, fileName, result, debugInfo, jsonData, rawText, imageBytes, c)
					if err != nil {
						log.Printf("Ошибка сохранения в БД: %v", err)
					}
				}
			}

			return true
		}
		return false
	}

	var clickItem = func(item config.Coordinates) {
		scripts.ClickCoordinates(port, c, item)
		combinedSaved := captureScreenShotsWithScroll()
		if !combinedSaved {
			saveScreenShot()
		}

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
	}

	var clickEveryItemAnsScreenShot = func(img image.Image) {
		// прокликиваем первую страницу
		points := imageInternal.FindItemPositionsByTextColor(img, 100)
		if len(points) > 2 {
			for _, point := range points {
				clickItem(config.Coordinates{Y: point.Y + marginY, X: marginX + point.X})
			}
		}

		// clickItem(config.Coordinates{X: marginX + c.Click.Item1.X, Y: marginY + c.Click.Item1.Y})
	}

	// берем в фокус и делаем скрин
	// scripts.ClickCoordinates(port, c, c.Click.Item1)
	// img = captureScreenShot()
	// clickEveryItemAnsScreenShot(img)

	// берем в фокус
	scripts.ClickCoordinates(port, c, c.Click.Item1)

	cycles := 0
	for cycles < 1 {
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

		// img = captureScreenShot()
		// SixButtonPx, _, _, _ := imageInternal.GetPixelColor(img, c.Click.Button6.X, 35)
		// maxSixButtonClicks := 0

		// for SixButtonPx > 30 && maxSixButtonClicks < 50 {
		// 	scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button6.X, Y: marginY + c.Click.Button6.Y})
		// 	img = captureScreenShot()
		// 	clickEveryItemAnsScreenShot(img)
		// 	img = captureScreenShot()
		// 	SixButtonPx, _, _, _ = imageInternal.GetPixelColor(img, c.Click.Button6.X, 35)
		// 	maxSixButtonClicks += 1
		// }

		// scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
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

	// Сохраняем структурированные данные
	if jsonData != "" {
		err = ocr.SaveStructuredData(db, int(ocrResultID), jsonData)
		if err != nil {
			log.Printf("Ошибка сохранения структурированных данных: %v", err)
		}
	}

	log.Printf("OCR результат и изображение сохранены в базу данных для файла: %s (ID: %d)", imagePath, ocrResultID)
	return int(ocrResultID), nil
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
