package scpript1

import (
	"image"
	"octopus/internal/click_manager"
	"octopus/internal/config"
	"octopus/internal/database"
	imageInternal "octopus/internal/image"
	"octopus/internal/logger"
	"octopus/internal/ocr"
	"octopus/internal/screenshot"
)

// clickPageButton кликает по кнопке
func clickPageButton(c *config.Config, clickManager *click_manager.ClickManager, dbManager *database.DatabaseManager, buttonName string, buttonCoords image.Point, isActive bool, marginX, marginY int, loggerManager *logger.LoggerManager) {
	if isActive {
		loggerManager.Info("🔘 Кликаем по %s...", buttonName)
		clickManager.ClickCoordinates(buttonCoords, marginX, marginY)
	} else {
		loggerManager.Info("⏭️ %s неактивен, пропускаем", buttonName)
	}
}

// getScreenshotOfItemPage делает скриншот страницы предмета
func getScreenshotOfItemPage(c *config.Config, clickManager *click_manager.ClickManager, screenshotManager *screenshot.ScreenshotManager, buttonStatus screenshot.ButtonStatus, scrollRPx int, marginX, marginY int, loggerManager *logger.LoggerManager) (image.Image, error) {
	// Если нет скролла, делаем обычный скриншот
	if scrollRPx <= 26 {
		loggerManager.Info("❌ Скролл не найден (scrollRPx <= 26), делаем обычный скриншот")
		return screenshotManager.CaptureScreenShot(), nil
	}

	// Выполняем основной цикл скриншотов и OCR (без нажатия кнопок)
	loggerManager.Info("🔄 Выполняем цикл скриншотов со скроллом...")
	if buttonStatus.Button2Active {
		img, _, err := clickManager.PerformScreenshotWithScroll(true)
		if err != nil {
			loggerManager.LogError(err, "Ошибка в основном цикле скриншотов со скроллом")
			return img, err
		}
		return img, nil
	} else {
		img, _, err := clickManager.PerformScreenshotWithScroll(false)
		if err != nil {
			loggerManager.LogError(err, "Ошибка в цикле скриншотов со скроллом")
			return img, err
		}
		return img, nil
	}
}

// processItemPages обрабатывает отдельный предмет (клик, проверка скролла, обработка скриншотов)
func processItemPages(c *config.Config, clickManager *click_manager.ClickManager, screenshotManager *screenshot.ScreenshotManager, dbManager *database.DatabaseManager, ocrManager *ocr.OCRManager, point image.Point, marginX, marginY int, loggerManager *logger.LoggerManager) {

	img := screenshotManager.CaptureScreenShot()

	// Сначала проверяем, есть ли скролл вообще
	scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
	loggerManager.Debug("scrollRPx: %v %v %v", scrollRPx, scrollGPx, scrollBPx)

	// Проверяем наличие всех кнопок
	loggerManager.Info("🔍 Проверяем наличие кнопок...")
	buttonStatus := screenshotManager.CheckAllButtonsStatus(img, c, marginX, marginY)

	// Обрабатываем страницу предмета
	itemPageImg, err := getScreenshotOfItemPage(c, clickManager, screenshotManager, buttonStatus, scrollRPx, marginX, marginY, loggerManager)
	if err != nil {
		loggerManager.LogError(err, "Ошибка получения скриншота")
		return
	}

	result, debugInfo, jsonData, rawText, err := ocrManager.ProcessImage(itemPageImg, "itemPageImg")
	if err != nil {
		loggerManager.LogError(err, "Ошибка OCR")
	}

	// Сохраняем результат в базу данных
	imageBytes, err := imageInternal.ImageToBytes(itemPageImg)
	if err != nil {
		loggerManager.LogError(err, "Ошибка конвертации изображения")
		return
	}

	_, err = dbManager.SaveOCRResultToDB("itemPageImg", result, debugInfo, jsonData, rawText, imageBytes, c)
	if err != nil {
		loggerManager.LogError(err, "Ошибка сохранения в БД")
	}

	// Кликаем Back только после последней существующей кнопки
	loggerManager.Info("🔙 Кликаем по кнопке Back...")
	clickManager.ClickCoordinates(image.Point{X: c.Click.Back.X, Y: c.Click.Back.Y}, marginX, marginY)
	loggerManager.Info("✅ Back клик выполнен")
}

var Run = func(c *config.Config, screenshotManager *screenshot.ScreenshotManager, dbManager *database.DatabaseManager, ocrManager *ocr.OCRManager, clickManager *click_manager.ClickManager, marginX, marginY int, loggerManager *logger.LoggerManager) {
	// Инициализация окна для получения отступов
	windowInitializer := imageInternal.NewWindowInitializer(c.WindowTopOffset)
	marginX, marginY, err := windowInitializer.GetItemBrokerWindowMargins()
	if err != nil {
		loggerManager.LogError(err, "Ошибка инициализации окна")
	}

	// берем окно L2 в фокус
	clickManager.FocusL2Window()

	// цикл обработки страниц
	for cycles := 0; cycles < c.MaxCyclesItemsList; cycles++ {
		loggerManager.Info("🔄 Обрабатываем страницу %d из %d", cycles+1, c.MaxCyclesItemsList)

		// обрабатываем первую страницу
		// получаем координаты всех предметов на странице
		itemCoordinates, err := screenshotManager.GetItemListItemsCoordinates()
		if err != nil {
			loggerManager.LogError(err, "Ошибка при поиске координат первой страницы")
		}

		// Обрабатываем каждый найденный предмет
		for _, coordinate := range itemCoordinates {
			loggerManager.Info("📍 Обрабатываем элемент в координатах: %v", coordinate)

			// кликаем по предмету
			clickManager.ClickCoordinates(coordinate, marginX, marginY)

			// сохраняем окно покупки в переменную
			img := screenshotManager.CaptureScreenShot()

			// определим есть ли кнопки на странице
			buttonStatus := screenshotManager.CheckAllButtonsStatus(img, c, marginX, marginY)
			if buttonStatus.Button2Active {
				loggerManager.Info("🔘 Кнопка 2 активна")
			} else {
				loggerManager.Info("⏭️ Кнопка 2 неактивна")
			}

			// определяем есть ли скролл на странице
			if screenshotManager.CheckScrollExists(img) {
				loggerManager.Info("✅ Скролл найден")
			} else {
				loggerManager.Info("❌ Скролл не найден (scrollRPx <= 26)")
			}

			// processItemPages(c, clickManager, screenshotManager, dbManager, ocrManager, coordinate, marginX, marginY, loggerManager)
		}

		loggerManager.Info("✅ Обработали все элементы на странице %d из %d", cycles+1, c.MaxCyclesItemsList)
	}
}
