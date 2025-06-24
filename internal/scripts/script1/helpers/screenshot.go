package helpers

import (
	"fmt"
	"image"
	"io/ioutil"
	"octopus/internal/config"
	"octopus/internal/screenshot"
)

// ScreenshotHelper содержит функции для работы со скриншотами
type ScreenshotHelper struct {
	marginX int
	marginY int
}

// NewScreenshotHelper создает новый экземпляр ScreenshotHelper
func NewScreenshotHelper(marginX, marginY int) *ScreenshotHelper {
	return &ScreenshotHelper{
		marginX: marginX,
		marginY: marginY,
	}
}

// CaptureScreenShot делает скриншот области
func (h *ScreenshotHelper) CaptureScreenShot() image.Image {
	img, _ := screenshot.CaptureScreenshot(config.CoordinatesWithSize{
		X:      h.marginX,
		Y:      h.marginY,
		Width:  300,
		Height: 361,
	})
	return img
}

// SaveScreenShot сохраняет скриншот в файл
func (h *ScreenshotHelper) SaveScreenShot(cfg *config.Config) image.Image {
	img, _ := screenshot.SaveScreenshot(config.CoordinatesWithSize{
		X:      h.marginX,
		Y:      h.marginY,
		Width:  300,
		Height: 361,
	}, cfg)
	return img
}

// SaveScreenShotFull сохраняет полный скриншот
func (h *ScreenshotHelper) SaveScreenShotFull() image.Image {
	img, _ := screenshot.SaveScreenshotFull(config.CoordinatesWithSize{
		X:      h.marginX,
		Y:      h.marginY,
		Width:  300,
		Height: 361,
	})
	return img
}

// CountFilesInDir подсчитывает количество файлов в директории
func CountFilesInDir(dir string) (int, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("не удалось прочитать папку: %v", err)
	}
	return len(files), nil
}
