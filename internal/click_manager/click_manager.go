package click_manager

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"github.com/tarm/serial"

	"octopus/internal/arduino"
	"octopus/internal/config"
	"octopus/internal/database"
	imageInternal "octopus/internal/image"
	"octopus/internal/screenshot"
)

// ScreenshotManager интерфейс для работы со скриншотами
type ScreenshotManager interface {
	CaptureScreenShot() image.Image
	SaveScreenShot(cfg *config.Config) image.Image
}

// ClickManager управляет кликами и скроллом
type ClickManager struct {
	port             *serial.Port
	config           *config.Config
	marginX          int
	marginY          int
	screenshotHelper ScreenshotManager
	imageHelper      *imageInternal.ImageHelper
	dbManager        *database.DatabaseManager
}

// NewClickManager создает новый экземпляр ClickManager
func NewClickManager(port *serial.Port, config *config.Config, marginX, marginY int, screenshotHelper ScreenshotManager, dbManager *database.DatabaseManager) *ClickManager {
	return &ClickManager{
		port:             port,
		config:           config,
		marginX:          marginX,
		marginY:          marginY,
		screenshotHelper: screenshotHelper,
		imageHelper:      imageInternal.NewImageHelper(port, config, marginX, marginY),
		dbManager:        dbManager,
	}
}

// CheckAndScreenScroll проверяет и выполняет скролл экрана
func (m *ClickManager) CheckAndScreenScroll(counter int, x int, img image.Image) (int, int) {
	scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
	fmt.Printf("scrollRPx: %v %v %v\n", scrollRPx, scrollGPx, scrollBPx)
	if scrollRPx > 26 {
		arduino.ScrollUp(m.port, m.config, counter+5)
		return counter + 1, x
	}
	return counter, x
}

// CheckAndClickScreenScroll проверяет и кликает по скроллу
func (m *ClickManager) CheckAndClickScreenScroll(counter int, img image.Image) (int, int) {
	scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
	fmt.Printf("scrollRPx: %v %v %v\n", scrollRPx, scrollGPx, scrollBPx)
	if scrollRPx > 26 {
		arduino.ClickCoordinates(m.port, m.config, config.Coordinates{X: m.marginX + 290, Y: m.marginY + 15})
		return counter + 1, 290
	}
	return counter, 290
}

// saveImage сохраняет изображение в файл
func (m *ClickManager) saveImage(img image.Image, fileName string) error {
	outFile, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer outFile.Close()

	err = png.Encode(outFile, img)
	if err != nil {
		return fmt.Errorf("failed to save image: %v", err)
	}
	return nil
}

// combineImagesVertically объединяет два изображения вертикально
func (m *ClickManager) combineImagesVertically(img1, img2 image.Image) (image.Image, error) {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	width := bounds1.Dx()
	height := bounds1.Dy() + bounds2.Dy()

	combinedImg := image.NewRGBA(image.Rect(0, 0, width, height))

	// Копируем первое изображение
	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			combinedImg.Set(x, y, img1.At(x, y))
		}
	}

	// Копируем второе изображение
	for y := bounds2.Min.Y; y < bounds2.Max.Y; y++ {
		for x := bounds2.Min.X; x < bounds2.Max.X; x++ {
			combinedImg.Set(x, y+bounds1.Dy(), img2.At(x, y))
		}
	}

	return combinedImg, nil
}

// PerformScreenshotWithScroll выполняет скриншот со скроллом
func (m *ClickManager) PerformScreenshotWithScroll(buttonPressed bool) (image.Image, string, error) {
	fmt.Println("=== Начало выполнения performScreenshotWithScroll ===")

	// Захватываем первый скриншот
	img := m.screenshotHelper.CaptureScreenShot()
	scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
	fmt.Printf("scrollRPx: %v %v %v\n", scrollRPx, scrollGPx, scrollBPx)

	// Если нет скролла, возвращаем первый скриншот
	if scrollRPx <= 26 {
		fmt.Println("❌ Скролл не найден (scrollRPx <= 26), возвращаем первый скриншот")
		fileCount, _ := screenshot.CountFilesInDir("./imgs")
		fileName := fmt.Sprintf("%s/screenshot_%d.png", "./imgs", fileCount)
		err := m.saveImage(img, fileName)
		if err != nil {
			return nil, "", err
		}
		return img, fileName, nil
	}

	fmt.Println("✅ Скролл найден, продолжаем выполнение")

	// Сохраняем первый скриншот
	fileCount, _ := screenshot.CountFilesInDir("./imgs")
	fileName1 := fmt.Sprintf("%s/screenshot_1_%d.png", "./imgs", fileCount)
	err := m.saveImage(img, fileName1)
	if err != nil {
		return nil, "", err
	}

	// Выполняем скролл
	fmt.Println("📜 Выполняем скролл...")
	arduino.ScrollUp(m.port, m.config, 5)

	// Ждем немного для анимации
	time.Sleep(500 * time.Millisecond)

	// Захватываем второй скриншот
	fmt.Println("📸 Захватываем второй скриншот...")
	img2 := m.screenshotHelper.CaptureScreenShot()

	// Сохраняем второй скриншот
	fileName2 := fmt.Sprintf("%s/screenshot_2_%d.png", "./imgs", fileCount)
	err = m.saveImage(img2, fileName2)
	if err != nil {
		return nil, "", err
	}

	// Объединяем изображения
	fmt.Println("🔗 Объединяем изображения...")
	combinedImg, err := m.combineImagesVertically(img, img2)
	if err != nil {
		return nil, "", err
	}

	// Обрезаем объединенное изображение, если была нажата кнопка
	if buttonPressed {
		fmt.Println("✂️ Обрезаем изображение (кнопка была нажата)...")
		bounds := combinedImg.Bounds()
		cropRect := image.Rect(0, 0, bounds.Dx(), bounds.Dy()-100)
		croppedCombinedImg := combinedImg.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(cropRect)
		combinedImg = croppedCombinedImg
	}

	// Сохраняем объединенное изображение
	fileName := fmt.Sprintf("%s/screenshot_combined_%d.png", "./imgs", fileCount)
	err = imageInternal.SaveCombinedImage(combinedImg, fileName)
	if err != nil {
		return nil, "", err
	}

	// Выполняем дополнительный скролл
	arduino.ScrollUp(m.port, m.config, 5)

	fmt.Println("=== Завершение performScreenshotWithScroll ===")
	return combinedImg, fileName, nil
}

// CaptureScreenShotsWithScroll выполняет захват скриншотов со скроллом

// ClickItem кликает по элементу и обрабатывает результат
func (m *ClickManager) ClickItem(item config.Coordinates) {

}

// GetPort возвращает порт для использования в других компонентах
func (m *ClickManager) GetPort() *serial.Port {
	return m.port
}

// FocusL2Window фокусирует окно L2, кликая по координатам Item1
func (m *ClickManager) FocusL2Window() {
	arduino.ClickCoordinates(m.port, m.config, m.config.Click.Item1)
}

// ClickCoordinates выполняет клик по указанным координатам
func (m *ClickManager) ClickCoordinates(coordinates config.Coordinates) {
	arduino.ClickCoordinates(m.port, m.config, coordinates)
}
