package screenshot

import (
	"database/sql"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"shnyr/internal/config"

	"shnyr/internal/helpers"

	"github.com/kbinani/screenshot"
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

		// TODO: Добавить прямую интеграцию OCR здесь
		// Вместо вызова exec.Command

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

// CaptureScreenShot делает скриншот области
func (h *ScreenshotManager) CaptureScreenShot() image.Image {
	img, _ := CaptureScreenshot(config.CoordinatesWithSize{
		X:      h.marginX,
		Y:      h.marginY,
		Width:  300,
		Height: 361,
	})
	return img
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
	img := h.CaptureScreenShot()
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
	Button2Active bool
	Button3Active bool
	Button4Active bool
	Button5Active bool
	Button6Active bool
}

// CheckButtonActive проверяет активность кнопки
func (h *ScreenshotManager) CheckButtonActive(buttonX, buttonY int, buttonName string, img image.Image) bool {
	buttonRPx, _, _, _ := helpers.GetPixelColor(img, buttonX, 36)
	return buttonRPx == 86
}

// CheckAllButtonsStatus проверяет статус всех кнопок на изображении
func (h *ScreenshotManager) CheckAllButtonsStatus(img image.Image, config *config.Config, marginX, marginY int) ButtonStatus {
	button2Active := h.CheckButtonActive(config.Click.Button2.X, config.Click.Button2.Y, "listButton2", img)
	button3Active := h.CheckButtonActive(config.Click.Button3.X, config.Click.Button3.Y, "listButton3", img)
	button4Active := h.CheckButtonActive(config.Click.Button4.X, config.Click.Button4.Y, "listButton4", img)
	button5Active := h.CheckButtonActive(config.Click.Button5.X, config.Click.Button5.Y, "listButton5", img)
	button6Active := h.CheckButtonActive(config.Click.Button6.X, config.Click.Button6.Y, "listButton6", img)

	return ButtonStatus{
		Button2Active: button2Active,
		Button3Active: button3Active,
		Button4Active: button4Active,
		Button5Active: button5Active,
		Button6Active: button6Active,
	}
}

// CheckScrollExists проверяет наличие скролла на изображении
func (h *ScreenshotManager) CheckScrollExists(img image.Image) bool {
	scrollRPx, _, _, _ := helpers.GetPixelColor(img, 290, 15)
	return scrollRPx > 26
}

// GetScrollInfo возвращает информацию о скролле (для отладки)
func (h *ScreenshotManager) GetScrollInfo(img image.Image) (int, int, int, error) {
	return helpers.GetPixelColor(img, 290, 15)
}

// PageStatus содержит полный статус страницы
type PageStatus struct {
	Buttons   ButtonStatus
	HasScroll bool
}

// GetPageStatus возвращает полный статус страницы (кнопки + скролл)
func (h *ScreenshotManager) GetPageStatus(img image.Image, config *config.Config, marginX, marginY int) PageStatus {
	buttons := h.CheckAllButtonsStatus(img, config, marginX, marginY)
	hasScroll := h.CheckScrollExists(img)

	return PageStatus{
		Buttons:   buttons,
		HasScroll: hasScroll,
	}
}
