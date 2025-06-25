package screenshot

import (
	"database/sql"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kbinani/screenshot"

	"shnyr/internal/arduino"
	"shnyr/internal/config"
	"shnyr/internal/helpers"
	"shnyr/internal/imageutils"
	"shnyr/internal/logger"
)

// Глобальная переменная для базы данных
var db *sql.DB

// SetDatabase устанавливает глобальную переменную базы данных
func SetDatabase(database *sql.DB) {
	db = database
}

// CaptureScreenshot захватывает скриншот в память и возвращает декодированное изображение
func CaptureScreenshot(c config.CoordinatesWithSize) (image.Image, error) {
	// Определяем область для захвата с переданными координатами
	bounds := image.Rect(c.X, c.Y, c.X+c.Width, c.Y+c.Height)

	// Захватываем экран в память
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %v", err)
	}

	return img, nil
}

// CaptureFullScreen захватывает скриншот всего экрана
func CaptureFullScreen() (image.Image, error) {
	// Захватываем весь экран
	img, err := screenshot.CaptureRect(image.Rect(0, 0, 800, 800)) // Стандартное разрешение, можно адаптировать
	if err != nil {
		return nil, fmt.Errorf("failed to capture full screen: %v", err)
	}
	return img, nil
}

func SaveScreenshot(c config.CoordinatesWithSize, cfg *config.Config) (image.Image, error) {
	// Получаем список файлов в папке ./imgs/
	files, err := filepath.Glob("./imgs/*")
	if err != nil {
		log.Println("Error reading files in ./imgs/:", err)
		return nil, err
	}

	// Количество файлов в папке
	screenshotCount := len(files)

	// Захватываем скриншот
	img, err := CaptureScreenshot(config.CoordinatesWithSize{X: c.X, Y: c.Y, Width: c.Width, Height: c.Height})
	if err != nil {
		log.Println("Error taking screenshot:", err)
		return nil, err
	}

	// --- Новая логика кадрирования ---
	// 40 пикселей слева, 22 пикселя сверху, 17 пикселей справа
	bounds := img.Bounds()
	cropRect := image.Rect(40, 22, bounds.Dx()-17, bounds.Dy())
	croppedImg := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(cropRect)
	// --- Конец логики кадрирования ---

	// Генерируем имя файла с номером, основанным на количестве файлов
	outputFile := fmt.Sprintf("./imgs/screenshot%d.png", screenshotCount+1)

	// Создаем файл для сохранения
	outFile, err := os.Create(outputFile)
	if err != nil {
		log.Println("Error creating file:", err)
		return nil, err
	}
	defer func(outFile *os.File) {
		err := outFile.Close()
		if err != nil {

		}
	}(outFile)

	// Сохраняем обрезанное изображение
	err = png.Encode(outFile, croppedImg)
	if err != nil {
		log.Println("Error saving image:", err)
		return nil, err
	} else {
		fmt.Println("Image saved:", outputFile)
		return croppedImg, nil
	}
}

var SaveItemOffersWithoutButtondScreenshot = func(c *config.Config) {
	SaveScreenshot(c.Screenshot.ItemOffersListWithoutButtons, c)
}

// SaveScreenshotFull захватывает и сохраняет скриншот указанной области без обрезки краёв для отладки
func SaveScreenshotFull(c config.CoordinatesWithSize) (image.Image, error) {
	// Получаем список файлов в папке ./imgs/
	files, err := filepath.Glob("./imgs/*")
	if err != nil {
		log.Println("Error reading files in ./imgs/:", err)
		return nil, err
	}

	// Количество файлов в папке
	screenshotCount := len(files)

	// Захватываем скриншот
	img, err := CaptureScreenshot(config.CoordinatesWithSize{X: c.X, Y: c.Y, Width: c.Width, Height: c.Height})
	if err != nil {
		log.Println("Error taking screenshot:", err)
		return nil, err
	}

	// Генерируем имя файла с номером, основанным на количестве файлов
	outputFile := fmt.Sprintf("./imgs/full_screenshot%d.png", screenshotCount+1)

	// Создаем файл для сохранения
	outFile, err := os.Create(outputFile)
	if err != nil {
		log.Println("Error creating file:", err)
		return nil, err
	}
	defer func(outFile *os.File) {
		err := outFile.Close()
		if err != nil {

		}
	}(outFile)

	// Сохраняем изображение без обрезки краёв
	err = png.Encode(outFile, img)
	if err != nil {
		log.Println("Error saving image:", err)
		return nil, err
	} else {
		fmt.Println("Full screenshot saved:", outputFile)
		return img, nil
	}
}

// ScreenshotManager содержит функции для работы со скриншотами
type ScreenshotManager struct {
	marginX int
	marginY int
}

// NewScreenshotManager создает новый экземпляр ScreenshotManager
func NewScreenshotManager(marginX, marginY int) *ScreenshotManager {
	return &ScreenshotManager{
		marginX: marginX,
		marginY: marginY,
	}
}

// checkImageQuality проверяет качество изображения по количеству пикселей с низкими значениями каналов
func (h *ScreenshotManager) checkImageQuality(img image.Image) bool {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	lowValuePixels := 0
	totalPixels := width * height

	// Проверяем каждый пиксель
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			// Проверяем все три канала на значение <= 26
			if r8 <= 26 && g8 <= 26 && b8 <= 26 {
				lowValuePixels++
			}
		}
	}

	// Вычисляем процент пикселей с низкими значениями
	lowValuePercentage := float64(lowValuePixels) / float64(totalPixels) * 100

	// Возвращаем true если пикселей с низкими значениями меньше 95%
	return lowValuePercentage < 99.9
}

// CaptureScreenShot делает скриншот области с проверкой качества
func (h *ScreenshotManager) CaptureScreenShot() (image.Image, error) {
	maxAttempts := 5

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		img, err := CaptureScreenshot(config.CoordinatesWithSize{
			X:      h.marginX,
			Y:      h.marginY,
			Width:  300,
			Height: 361,
		})

		if err != nil {
			log.Printf("Ошибка захвата скриншота (попытка %d/%d): %v", attempt, maxAttempts, err)
			if attempt == maxAttempts {
				return nil, fmt.Errorf("не удалось захватить скриншот после %d попыток: %v", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// Проверяем качество изображения
		if h.checkImageQuality(img) {
			return img, nil
		}

		log.Printf("Плохое качество скриншота (попытка %d/%d), ожидание 1 секунды...", attempt, maxAttempts)

		if attempt < maxAttempts {
			time.Sleep(1 * time.Second)
		}
	}

	return nil, fmt.Errorf("не удалось получить качественный скриншот после %d попыток", maxAttempts)
}

// SaveScreenShot сохраняет скриншот в файл
func (h *ScreenshotManager) SaveScreenShot(cfg *config.Config) image.Image {
	img, _ := SaveScreenshot(config.CoordinatesWithSize{
		X:      h.marginX,
		Y:      h.marginY,
		Width:  300,
		Height: 361,
	}, cfg)
	return img
}

// SaveScreenShotFull сохраняет полный скриншот
func (h *ScreenshotManager) SaveScreenShotFull() image.Image {
	img, _ := SaveScreenshotFull(config.CoordinatesWithSize{
		X:      h.marginX,
		Y:      h.marginY,
		Width:  300,
		Height: 361,
	})
	return img
}

// CountFilesInDir подсчитывает количество файлов в директории
func CountFilesInDir(dir string) (int, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return 0, fmt.Errorf("не удалось прочитать папку: %v", err)
	}
	return len(files), nil
}

// GetCoordinatesItemsInItemList возвращает массив точек для клика на изображении
func (h *ScreenshotManager) GetCoordinatesItemsInItemList(img image.Image) ([]image.Point, error) {
	points := h.findItemPositionsByTextColor(img, 80)
	if len(points) > 0 {
		return points, nil
	} else {
		return nil, fmt.Errorf("недостаточно точек для обработки (нужно > 0, найдено: %d)", len(points))
	}
}

// GetItemListItemsCoordinates ищет координаты предметов на странице списка предметов
func (h *ScreenshotManager) GetItemListItemsCoordinates() ([]image.Point, error) {
	img, err := h.CaptureScreenShot()
	if err != nil {
		return nil, err
	}

	coordinates, err := h.GetCoordinatesItemsInItemList(img)
	if err != nil {
		return nil, err
	}

	return coordinates, nil
}

// findItemPositionsByTextColor находит центры цветных строк с названиями предметов
func (h *ScreenshotManager) findItemPositionsByTextColor(img image.Image, targetX int) []image.Point {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	type bar struct{ yStart, yEnd int }
	var allBars []bar

	// --- Этап 1: Находим все отдельные строки цветного текста ---
	const scanXStart = 70
	const minHorizontalPixels = 20
	const colorThreshold = 20

	inBar := false
	var barYStart int
	for y := 30; y < height; y++ {
		activePixelCount := 0
		for x := scanXStart; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			isGreen := g8 > r8+colorThreshold && g8 > b8+colorThreshold
			isRed := r8 > g8+colorThreshold && r8 > b8+colorThreshold
			if isGreen || isRed {
				activePixelCount++
			}
		}

		isRowActive := activePixelCount >= minHorizontalPixels
		if isRowActive && !inBar {
			inBar = true
			barYStart = y
		} else if !isRowActive && inBar {
			inBar = false
			allBars = append(allBars, bar{yStart: barYStart, yEnd: y - 1})
		}
	}
	if inBar {
		allBars = append(allBars, bar{yStart: barYStart, yEnd: height - 1})
	}

	// --- Этап 2: Группируем близкие строки и вычисляем центры ---
	var centers []image.Point
	if len(allBars) == 0 {
		return centers
	}

	const minDistanceY = 15 // Макс. расстояние для объединения в одну группу.
	currentGroup := allBars[0]

	for i := 1; i < len(allBars); i++ {
		nextBar := allBars[i]
		// Если следующая строка близко, она является частью текущей группы.
		if (nextBar.yStart - currentGroup.yEnd) < minDistanceY {
			// Расширяем границы группы.
			currentGroup.yEnd = nextBar.yEnd
		} else {
			// Следующая строка далеко - значит, предыдущая группа закончилась.
			// Вычисляем и сохраняем ее центр.
			centerY := currentGroup.yStart + (currentGroup.yEnd-currentGroup.yStart)/2
			centers = append(centers, image.Point{X: targetX, Y: centerY})
			// Начинаем новую группу.
			currentGroup = nextBar
		}
	}

	// Сохраняем центр последней группы.
	lastCenterY := currentGroup.yStart + (currentGroup.yEnd-currentGroup.yStart)/2
	centers = append(centers, image.Point{X: targetX, Y: lastCenterY})

	return centers
}

// ButtonStatus содержит статус всех кнопок
type ButtonStatus struct {
	Button1Active bool
	Button2Active bool
	Button3Active bool
	Button4Active bool
}

// CheckButtonActive проверяет активность кнопки
func (h *ScreenshotManager) CheckButtonActive(buttonX, buttonY int, buttonName string, img image.Image) bool {
	buttonRPx, _, _, _ := helpers.GetPixelColor(img, buttonX, 36)
	return buttonRPx == 86
}

// CheckAllButtonsStatus проверяет статус всех кнопок на изображении
func (h *ScreenshotManager) CheckAllButtonsStatus(img image.Image, config *config.Config, marginX, marginY int) ButtonStatus {
	button1Active := h.CheckButtonActive(config.Click.Button1.X, config.ListButtonBottomYCoordinate, "listButton1", img)
	button2Active := h.CheckButtonActive(config.Click.Button2.X, config.ListButtonBottomYCoordinate, "listButton2", img)
	button3Active := h.CheckButtonActive(config.Click.Button3.X, config.ListButtonBottomYCoordinate, "listButton3", img)
	button4Active := h.CheckButtonActive(config.Click.Button4.X, config.ListButtonBottomYCoordinate, "listButton4", img)

	return ButtonStatus{
		Button1Active: button1Active,
		Button2Active: button2Active,
		Button3Active: button3Active,
		Button4Active: button4Active,
	}
}

// CheckScrollExists проверяет наличие скролла на изображении
func (h *ScreenshotManager) CheckScrollExists() bool {
	img, err := h.CaptureScreenShot()
	if err != nil {
		return false
	}

	scrollRPx, _, _, _ := helpers.GetPixelColor(img, 290, 15)
	return scrollRPx > 26
}

// PageStatus содержит полный статус страницы
type PageStatus struct {
	Buttons   ButtonStatus
	HasScroll bool
}

// GetPageStatus возвращает полный статус страницы (кнопки + скролл)
func (h *ScreenshotManager) GetPageStatus(config *config.Config) PageStatus {
	img, err := h.CaptureScreenShot()
	if err != nil {
		// Возвращаем пустой статус если не удалось получить скриншот
		return PageStatus{
			Buttons:   ButtonStatus{},
			HasScroll: false,
		}
	}

	buttons := h.CheckAllButtonsStatus(img, config, h.marginX, h.marginY)
	hasScroll := h.CheckScrollExists()
	return PageStatus{
		Buttons:   buttons,
		HasScroll: hasScroll,
	}
}

// checkScrollByCoordinates проверяет скролл по указанным координатам
func (h *ScreenshotManager) checkScrollByCoordinates(x, y int) bool {
	img, err := h.CaptureScreenShot()
	if err != nil {
		return false
	}

	r, _, _, _ := helpers.GetPixelColor(img, x, y)
	return r > 26
}

// PerformScreenshotWithScroll выполняет скриншот со скроллом
func (h *ScreenshotManager) PerformScreenshotWithScroll(pageStatus PageStatus, config *config.Config) (image.Image, error) {
	// Списки для хранения всех скриншотов
	var screenshots []image.Image
	var smallScreenshots []image.Image

	img, err := h.CaptureScreenShot()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить качественный скриншот")
	}
	screenshots = append(screenshots, img)
	log.Printf("📸 Начальный скриншот захвачен")

	// создаем переменные scrollToBottom и clickToBottom
	scrollToBottom := h.checkScrollByCoordinates(config.ScrollBottomCheckPixelX, config.ScrollBottomCheckPixelYScroll)
	clickToBottom := false

	// создаем переменные scrollCounter и clickCounter для скролла вверх и ограничений на количество кликов и скролла вниз
	scrollCounter := 0
	clickCounter := 0

	// пока scrollToBottom не станет true, скроллим вниз
	for !scrollToBottom {
		arduino.ScrollDown(config, 1)
		img, err := h.CaptureScreenShot()
		if err != nil {
			return nil, fmt.Errorf("не удалось получить качественный скриншот во время скролла")
		}
		screenshots = append(screenshots, img)
		scrollToBottom = h.checkScrollByCoordinates(config.ScrollBottomCheckPixelX, config.ScrollBottomCheckPixelYScroll)
		scrollCounter++
		if scrollCounter > 40 {
			scrollToBottom = true
		}
	}

	// кликаем по скроллу чтобы перенести мышку на скролл
	arduino.ClickCoordinates(config, image.Point{X: h.marginX + config.Click.Scroll.X, Y: h.marginY + config.Click.Scroll.Y})
	clickCounter++

	// пока clickToBottom не станет true, кликаем по скроллу
	for !clickToBottom {
		arduino.FastClick(config)
		img, err := h.CaptureScreenShot()
		if err != nil {
			return nil, fmt.Errorf("не удалось получить качественный скриншот во время кликов")
		}
		clickToBottom = h.checkScrollByCoordinates(config.ScrollBottomCheckPixelX, config.ScrollBottomCheckPixelYClick)
		clickCounter++
		smallScreenshots = append(smallScreenshots, img)
		if clickCounter > 10 {
			clickToBottom = true
		}
	}

	// делаем в цикле скроллы наверх как сумма clickCounter и scrollCounter
	totalScrollsUp := clickCounter + scrollCounter + 5
	arduino.ScrollUp(config, totalScrollsUp)
	arduino.ScrollUp(config, 1)

	var finalImage image.Image

	if len(smallScreenshots) >= 2 {
		prev := smallScreenshots[len(smallScreenshots)-2]
		last := smallScreenshots[len(smallScreenshots)-1]
		diff, err := imageutils.LastColorStripeDistanceDiff(prev, last, 26, 20)
		if err != nil {
			return nil, err
		} else {
			finalImage, _ = imageutils.CombineImages(screenshots, smallScreenshots[:len(smallScreenshots)-1], smallScreenshots[len(smallScreenshots)-1], diff)
		}
	} else if len(smallScreenshots) == 1 {
		prev := screenshots[len(screenshots)-1]
		last := smallScreenshots[0]
		diff, err := imageutils.LastColorStripeDistanceDiff(prev, last, 26, 20)
		if err != nil {
			return nil, err
		} else {
			finalImage, _ = imageutils.CombineImages(screenshots, nil, smallScreenshots[0], diff)
		}
	} else {
		finalImage, _ = imageutils.CombineImages(screenshots, nil, nil, 0)
	}

	combinedImg := imageutils.CropOpacityPixel(finalImage)
	return combinedImg, nil
}

// SaveImage сохраняет изображение в папку imgs и возвращает название и путь к файлу
func (h *ScreenshotManager) SaveImage(img image.Image, filename string, saveAllScreenshots int, loggerManager *logger.LoggerManager) (string, error) {
	var finalFilename string

	if saveAllScreenshots == 1 {
		// Генерируем уникальное имя файла
		files, err := filepath.Glob("./imgs/*")
		if err != nil {
			return "", fmt.Errorf("ошибка чтения папки imgs: %v", err)
		}
		screenshotCount := len(files)
		finalFilename = fmt.Sprintf("%s_%d.png", strings.TrimSuffix(filename, ".png"), screenshotCount+1)
	} else {
		// Всегда перезаписываем файл
		finalFilename = filename
	}

	// Создаем файл
	f, err := os.Create("./imgs/" + finalFilename)
	if err != nil {
		return "", fmt.Errorf("ошибка создания файла: %v", err)
	}
	defer f.Close()

	// Сохраняем изображение в PNG формате
	err = png.Encode(f, img)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения изображения: %v", err)
	}

	// Возвращаем полный путь к файлу
	fullPath := "./imgs/" + finalFilename
	return fullPath, nil
}

// SaveScreenshot делает скриншот и сохраняет его как debug_screenshot.png
func (h *ScreenshotManager) SaveScreenshot() error {
	img, err := h.CaptureScreenShot()
	if err != nil {
		return fmt.Errorf("не удалось получить качественный скриншот")
	}

	// Создаем файл
	f, err := os.Create("./imgs/debug_screenshot.png")
	if err != nil {
		return fmt.Errorf("ошибка создания файла: %v", err)
	}
	defer f.Close()

	// Сохраняем изображение в PNG формате
	err = png.Encode(f, img)
	if err != nil {
		return fmt.Errorf("ошибка сохранения изображения: %v", err)
	}

	return nil
}

// CropImageForText обрезает изображение для кнопки "назад" с учетом статуса кнопок
func (h *ScreenshotManager) CropImageForText(img image.Image, config *config.Config, Button2Active bool) image.Image {
	bounds := img.Bounds()
	topCrop := config.BackButtonImageCropHeight // Используем значение из конфигурации
	if Button2Active {
		topCrop = config.BackButtonWithListButtonsImageCropHeight
	}

	fmt.Println("topCrop", topCrop)

	// обрезаем изображение
	cropRect := image.Rect(config.ItemsImgsWidth, topCrop, bounds.Dx()-config.ScrollWidth, bounds.Dy())
	croppedImg := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(cropRect)

	return croppedImg
}

// CheckButtonActiveByPixel проверяет активность кнопки по пикселю
func (h *ScreenshotManager) CheckButtonActiveByPixel(x, y int) bool {
	img, err := h.CaptureScreenShot()
	if err != nil {
		return false
	}

	r, _, _, _ := helpers.GetPixelColor(img, x, y)
	return r > 26
}
