package cycle_listed_items

import (
	"image"
	"shnyr/internal/click_manager"
	"shnyr/internal/config"
	"shnyr/internal/database"
	"shnyr/internal/interrupt"
	"shnyr/internal/logger"
	"shnyr/internal/ocr"
	"shnyr/internal/screenshot"
)

var Run = func(c *config.Config, screenshotManager *screenshot.ScreenshotManager, dbManager *database.DatabaseManager, ocrManager *ocr.OCRManager, clickManager *click_manager.ClickManager, loggerManager *logger.LoggerManager, interruptManager *interrupt.InterruptManager) {
	// Инициализируем таблицу предметов
	err := dbManager.InitializeItemsTable()
	if err != nil {
		loggerManager.LogError(err, "Ошибка инициализации таблицы предметов")
		return
	}

	// берем окно L2 в фокус
	clickManager.FocusL2Window()

	// Шнырь жмет F12 для открытия окна с предметами
	clickManager.F12()

	for cycles := 0; cycles < c.MaxCyclesItemsList; cycles++ {
		interrupted, _, checkErr := checkInterruption(interruptManager, dbManager, loggerManager)
		if checkErr != nil {
			loggerManager.Info("⏹️ Прерывание по запросу пользователя")
			return
		}
		if interrupted {
			loggerManager.Info("⏹️ Прерывание по запросу пользователя")
			return
		}

		loggerManager.Info("🔄 Проход %d из %d", cycles+1, c.MaxCyclesItemsList)

		// ОБРАБОТКА РАЗДЕЛА СКУПКИ (BUY)
		loggerManager.Info("💰 Начинаем обработку раздела СКУПКА (BUY)")

		// Кликаем на координаты 10, 10 для перехода в раздел buy
		clickManager.ClickCoordinates(image.Point{X: 53, Y: 46})
		loggerManager.Info("📍 Переходим в раздел скупки (координаты 53, 46)")

		// Обрабатываем все предметы для скупки (buy_consumables и buy_equipment)
		err = processItemsByCategory(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, "buy_consumables", cycles, c.StartButtonIndex)
		if err != nil {
			if err.Error() == "прерывание по запросу пользователя" {
				loggerManager.Info("⏹️ Завершение работы по прерыванию")
				return
			}
			loggerManager.LogError(err, "Ошибка при обработке предметов buy_consumables")
		}

		err = processItemsByCategory(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, "buy_equipment", cycles, c.StartButtonIndex)
		if err != nil {
			if err.Error() == "прерывание по запросу пользователя" {
				loggerManager.Info("⏹️ Завершение работы по прерыванию")
				return
			}
			loggerManager.LogError(err, "Ошибка при обработке предметов buy_equipment")
		}

		// ОБРАБОТКА РАЗДЕЛА ПРОДАЖИ (SELL)
		loggerManager.Info("💸 Начинаем обработку раздела ПРОДАЖА (SELL)")

		// Кликаем на координаты 15, 265 для перехода между разделами
		clickManager.ClickCoordinates(image.Point{X: 15, Y: 265})
		loggerManager.Info("📍 Переходим между разделами (координаты 15, 265)")

		// Кликаем на координаты 53, 64 для перехода в раздел sell
		clickManager.ClickCoordinates(image.Point{X: 53, Y: 64})
		loggerManager.Info("📍 Переходим в раздел продажи (координаты 53, 64)")

		// Обрабатываем все предметы для продажи (sell_consumables и sell_equipment)
		err = processItemsByCategory(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, "sell_consumables", cycles, c.StartButtonIndex)
		if err != nil {
			if err.Error() == "прерывание по запросу пользователя" {
				loggerManager.Info("⏹️ Завершение работы по прерыванию")
				return
			}
			loggerManager.LogError(err, "Ошибка при обработке предметов sell_consumables")
		}

		err = processItemsByCategory(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, "sell_equipment", cycles, c.StartButtonIndex)
		if err != nil {
			if err.Error() == "прерывание по запросу пользователя" {
				loggerManager.Info("⏹️ Завершение работы по прерыванию")
				return
			}
			loggerManager.LogError(err, "Ошибка при обработке предметов sell_equipment")
		}

		// Кликаем на координаты 15, 265 для перехода между разделами
		clickManager.ClickCoordinates(image.Point{X: 15, Y: 265})
		loggerManager.Info("📍 Переходим между разделами (координаты 15, 265)")

		loggerManager.Info("✅ Завершен проход %d", cycles+1)
	}

	loggerManager.Info("🎉 Все проходы завершены")
}
