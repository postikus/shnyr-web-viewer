package helpers

import (
	"fmt"
	"image"
	"octopus/internal/config"
	imageInternal "octopus/internal/image"
	"octopus/internal/scripts"

	"github.com/tarm/serial"
)

// ButtonProcessor содержит функции для обработки кнопок
type ButtonProcessor struct {
	port             *serial.Port
	config           *config.Config
	marginX          int
	marginY          int
	screenshotHelper *ScreenshotHelper
	imageHelper      *ImageHelper
	ocrProcessor     *OCRProcessor
}

// NewButtonProcessor создает новый экземпляр ButtonProcessor
func NewButtonProcessor(port *serial.Port, config *config.Config, marginX, marginY int, ocrProcessor *OCRProcessor) *ButtonProcessor {
	screenshotHelper := NewScreenshotHelper(marginX, marginY)
	imageHelper := NewImageHelper(port, config, marginX, marginY)

	return &ButtonProcessor{
		port:             port,
		config:           config,
		marginX:          marginX,
		marginY:          marginY,
		screenshotHelper: screenshotHelper,
		imageHelper:      imageHelper,
		ocrProcessor:     ocrProcessor,
	}
}

// CaptureScreenShotsWithScroll выполняет захват скриншотов со скроллом
func (p *ButtonProcessor) CaptureScreenShotsWithScroll() bool {
	fmt.Println("=== Начало выполнения captureScreenShotsWithScroll ===")

	// Сначала проверяем, есть ли скролл вообще
	img := p.screenshotHelper.CaptureScreenShot()
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
	button2Active := p.imageHelper.CheckButtonActive(p.config.Click.Button2.X, p.config.Click.Button2.Y, "listButton2", img)
	button3Active := p.imageHelper.CheckButtonActive(p.config.Click.Button3.X, p.config.Click.Button3.Y, "listButton3", img)
	button4Active := p.imageHelper.CheckButtonActive(p.config.Click.Button4.X, p.config.Click.Button4.Y, "listButton4", img)
	button5Active := p.imageHelper.CheckButtonActive(p.config.Click.Button5.X, p.config.Click.Button5.Y, "listButton5", img)
	button6Active := p.imageHelper.CheckButtonActive(p.config.Click.Button6.X, p.config.Click.Button6.Y, "listButton6", img)

	fmt.Printf("📋 Статус кнопок: Button2=%v, Button3=%v, Button4=%v, Button5=%v, Button6=%v\n",
		button2Active, button3Active, button4Active, button5Active, button6Active)

	// Выполняем основной цикл скриншотов и OCR (без нажатия кнопок)
	fmt.Println("🔄 Выполняем основной цикл скриншотов и OCR...")
	if button2Active {
		err := p.ocrProcessor.PerformScreenshotAndOCR(true)
		if err != nil {
			fmt.Printf("❌ Ошибка в основном цикле: %v\n", err)
			return false
		}
	} else {
		err := p.ocrProcessor.PerformScreenshotAndOCR(false)
		if err != nil {
			fmt.Printf("❌ Ошибка в основном цикле: %v\n", err)
			return false
		}
	}

	fmt.Println("✅ Основной цикл выполнен успешно")

	// Идем по кнопкам последовательно
	if button2Active {
		fmt.Println("🔘 Кликаем по Button2...")
		scripts.ClickCoordinates(p.port, p.config, config.Coordinates{X: p.marginX + p.config.Click.Button2.X, Y: p.marginY + p.config.Click.Button2.Y})
		err := p.ocrProcessor.PerformScreenshotAndOCR(true)
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
		scripts.ClickCoordinates(p.port, p.config, config.Coordinates{X: p.marginX + p.config.Click.Button3.X, Y: p.marginY + p.config.Click.Button3.Y})
		err := p.ocrProcessor.PerformScreenshotAndOCR(true)
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
		scripts.ClickCoordinates(p.port, p.config, config.Coordinates{X: p.marginX + p.config.Click.Button4.X, Y: p.marginY + p.config.Click.Button4.Y})
		err := p.ocrProcessor.PerformScreenshotAndOCR(true)
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
		scripts.ClickCoordinates(p.port, p.config, config.Coordinates{X: p.marginX + p.config.Click.Button5.X, Y: p.marginY + p.config.Click.Button5.Y})
		err := p.ocrProcessor.PerformScreenshotAndOCR(true)
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
		scripts.ClickCoordinates(p.port, p.config, config.Coordinates{X: p.marginX + p.config.Click.Button6.X, Y: p.marginY + p.config.Click.Button6.Y})
		err := p.ocrProcessor.PerformScreenshotAndOCR(true)
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
	scripts.ClickCoordinates(p.port, p.config, config.Coordinates{X: p.marginX + p.config.Click.Back.X, Y: p.marginY + p.config.Click.Back.Y})
	fmt.Println("✅ Back клик выполнен")

	fmt.Println("=== Завершение captureScreenShotsWithScroll ===")
	return true
}

// ClickItem кликает по элементу и обрабатывает результат
func (p *ButtonProcessor) ClickItem(item config.Coordinates) {
	scripts.ClickCoordinates(p.port, p.config, item)
	combinedSaved := p.CaptureScreenShotsWithScroll()
	if !combinedSaved {
		p.screenshotHelper.SaveScreenShot(p.config)
		scripts.ClickCoordinates(p.port, p.config, config.Coordinates{X: p.marginX + p.config.Click.Back.X, Y: p.marginY + p.config.Click.Back.Y})
	}
}

// ClickEveryItemAndScreenShot кликает по всем элементам на изображении
func (p *ButtonProcessor) ClickEveryItemAndScreenShot(img image.Image) {
	// прокликиваем первую страницу
	points := imageInternal.FindItemPositionsByTextColor(img, 80)
	fmt.Printf("🔍 Найдено точек для клика: %d\n", len(points))
	if len(points) > 0 {
		fmt.Printf("✅ Найдено достаточно точек, начинаем обработку...\n")
		for i, point := range points {
			fmt.Printf("🖱️ Кликаем по точке %d: (%d, %d)\n", i+1, point.X, point.Y)
			p.ClickItem(config.Coordinates{Y: point.Y + p.marginY, X: p.marginX + point.X})
		}
	} else {
		fmt.Printf("⚠️ Недостаточно точек для обработки (нужно > 0, найдено: %d)\n", len(points))
	}
}
