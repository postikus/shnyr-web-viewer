package scpript1

import (
	"database/sql"
	"log"
	"octopus/internal/config"
	imageInternal "octopus/internal/image"
	"octopus/internal/scripts"
	"octopus/internal/scripts/script1/helpers"

	"github.com/tarm/serial"
)

var Run = func(port *serial.Port, c *config.Config, db *sql.DB) {
	// Инициализируем окно
	windowInitializer := helpers.NewWindowInitializer(23)
	marginX, marginY, err := windowInitializer.InitializeWindow()
	if err != nil {
		log.Fatalf("Ошибка инициализации окна: %v", err)
	}

	// Инициализируем вспомогательные компоненты
	screenshotHelper := helpers.NewScreenshotHelper(marginX, marginY)
	dbHelper := helpers.NewDatabaseHelper(db)

	// Инициализируем процессоры
	ocrProcessor := helpers.NewOCRProcessor(port, c, marginX, marginY, dbHelper)
	buttonProcessor := helpers.NewButtonProcessor(port, c, marginX, marginY, ocrProcessor)

	// берем в фокус
	scripts.ClickCoordinates(port, c, c.Click.Item1)

	cycles := 0
	for cycles < 20 {
		img := screenshotHelper.CaptureScreenShot()
		buttonProcessor.ClickEveryItemAndScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button2.X, Y: marginY + c.Click.Button2.Y})
		img = screenshotHelper.CaptureScreenShot()
		buttonProcessor.ClickEveryItemAndScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button3.X, Y: marginY + c.Click.Button3.Y})
		img = screenshotHelper.CaptureScreenShot()
		buttonProcessor.ClickEveryItemAndScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button4.X, Y: marginY + c.Click.Button4.Y})
		img = screenshotHelper.CaptureScreenShot()
		buttonProcessor.ClickEveryItemAndScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button5.X, Y: marginY + c.Click.Button5.Y})
		img = screenshotHelper.CaptureScreenShot()
		buttonProcessor.ClickEveryItemAndScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button6.X, Y: marginY + c.Click.Button6.Y})
		img = screenshotHelper.CaptureScreenShot()
		buttonProcessor.ClickEveryItemAndScreenShot(img)

		img = screenshotHelper.CaptureScreenShot()
		SixButtonPx, _, _, _ := imageInternal.GetPixelColor(img, c.Click.Button6.X, 35)
		maxSixButtonClicks := 0

		for SixButtonPx > 30 && maxSixButtonClicks < 50 {
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button6.X, Y: marginY + c.Click.Button6.Y})
			img = screenshotHelper.CaptureScreenShot()
			buttonProcessor.ClickEveryItemAndScreenShot(img)
			img = screenshotHelper.CaptureScreenShot()
			SixButtonPx, _, _, _ = imageInternal.GetPixelColor(img, c.Click.Button6.X, 35)
			maxSixButtonClicks += 1
		}

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})

		cycles += 1
	}
}
