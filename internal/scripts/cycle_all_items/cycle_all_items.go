package cycle_all_items

import (
	"image"
	"shnyr/internal/click_manager"
	"shnyr/internal/config"
	"shnyr/internal/database"
	imageInternal "shnyr/internal/image"
	"shnyr/internal/interrupt"
	"shnyr/internal/logger"
	"shnyr/internal/ocr"
	"shnyr/internal/screenshot"
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
func getScreenshotOfItemPage(c *config.Config, clickManager *click_manager.ClickManager, screenshotManager *screenshot.ScreenshotManager, buttonStatus screenshot.ButtonStatus, hasScroll bool, marginX, marginY int, loggerManager *logger.LoggerManager) (image.Image, error) {
	// Если нет скролла, делаем обычный скриншот
	if !hasScroll {
		loggerManager.Info("❌ Скролл не найден, делаем обычный скриншот")
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

	// Получаем полный статус страницы
	pageStatus := screenshotManager.GetPageStatus(img, c, marginX, marginY)
	loggerManager.Debug("scrollRPx: %v", pageStatus.HasScroll)

	// выводим общий лог статуса страницы
	loggerManager.Info("📄 Статус страницы: кнопка2=%v, кнопка3=%v, кнопка4=%v, кнопка5=%v, кнопка6=%v, скролл=%v",
		pageStatus.Buttons.Button2Active,
		pageStatus.Buttons.Button3Active,
		pageStatus.Buttons.Button4Active,
		pageStatus.Buttons.Button5Active,
		pageStatus.Buttons.Button6Active,
		pageStatus.HasScroll)

	// Обрабатываем страницу предмета
	itemPageImg, err := getScreenshotOfItemPage(c, clickManager, screenshotManager, pageStatus.Buttons, pageStatus.HasScroll, marginX, marginY, loggerManager)
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

var Run = func(c *config.Config, screenshotManager *screenshot.ScreenshotManager, dbManager *database.DatabaseManager, ocrManager *ocr.OCRManager, clickManager *click_manager.ClickManager, marginX, marginY int, loggerManager *logger.LoggerManager, interruptManager *interrupt.InterruptManager) {
	// Инициализация окна для получения отступов
	windowInitializer := imageInternal.NewWindowInitializer(c.WindowTopOffset)
	marginX, marginY, err := windowInitializer.GetItemBrokerWindowMargins()
	if err != nil {
		loggerManager.LogError(err, "Ошибка инициализации окна")
	}

	// берем окно L2 в фокус
	clickManager.FocusL2Window()

	// цикл обработки страниц с предметами, количество полных проходов хранится в конфиге в переменной max_cycles_items_list
	for cycles := 0; cycles < c.MaxCyclesItemsList; cycles++ {
		loggerManager.Info("🔄 Проход %d из %d", cycles+1, c.MaxCyclesItemsList)

		// обрабатываем первую страницу
		// получаем координаты всех предметов на странице
		itemCoordinates, err := screenshotManager.GetItemListItemsCoordinates()
		if err != nil {
			loggerManager.LogError(err, "Ошибка при поиске координат первой страницы")
		}

		// Обрабатываем каждый найденный предмет
		for _, coordinate := range itemCoordinates {
			// Проверяем сигнал прерывания в начале обработки каждого предмета
			select {
			case <-interruptManager.GetScriptInterruptChan():
				loggerManager.Info("⏹️ Прерывание cycle_all_items по запросу пользователя")
				return
			default:
			}

			loggerManager.Info("📍 Обрабатываем элемент в координатах: %v", coordinate)

			// кликаем по предмету
			clickManager.ClickCoordinates(coordinate, marginX, marginY)

			// сохраняем окно покупки в переменную
			img := screenshotManager.CaptureScreenShot()

			// получаем полный статус страницы
			pageStatus := screenshotManager.GetPageStatus(img, c, marginX, marginY)

			// выводим общий лог статуса страницы
			loggerManager.Info("📄 Статус страницы: кнопка2=%v, кнопка3=%v, кнопка4=%v, кнопка5=%v, кнопка6=%v, скролл=%v",
				pageStatus.Buttons.Button2Active,
				pageStatus.Buttons.Button3Active,
				pageStatus.Buttons.Button4Active,
				pageStatus.Buttons.Button5Active,
				pageStatus.Buttons.Button6Active,
				pageStatus.HasScroll)

			// processItemPages(c, clickManager, screenshotManager, dbManager, ocrManager, coordinate, marginX, marginY, loggerManager)

			// кликаем по back
			clickManager.ClickCoordinates(image.Point{X: c.Click.Back.X, Y: c.Click.Back.Y}, marginX, marginY)
		}

		loggerManager.Info("✅ Обработали все элементы на странице %d из %d", cycles+1, c.MaxCyclesItemsList)
	}
}
