package helpers

import (
	"fmt"
	"image"
	imageInternal "octopus/internal/image"
	"octopus/internal/screen"
)

// WindowInitializer содержит функции для инициализации окна
type WindowInitializer struct {
	topOffset int
}

// NewWindowInitializer создает новый экземпляр WindowInitializer
func NewWindowInitializer(topOffset int) *WindowInitializer {
	return &WindowInitializer{
		topOffset: topOffset,
	}
}

// InitializeWindow инициализирует окно и возвращает координаты
func (w *WindowInitializer) InitializeWindow() (int, int, error) {
	// Делаем скриншот всего экрана
	img, err := screen.CaptureFullScreen()
	if err != nil {
		return 0, 0, fmt.Errorf("ошибка захвата экрана: %v", err)
	}

	// Обрезаем верхние пиксели
	bounds := img.Bounds()
	croppedImg := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()-w.topOffset))
	for y := w.topOffset; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			croppedImg.Set(x, y-w.topOffset, img.At(x, y))
		}
	}

	// Ищем окно
	gameWindow, err := imageInternal.FindGameWindow(croppedImg)
	if err != nil {
		return 0, 0, fmt.Errorf("окно не найдено: %v", err)
	}

	marginX := gameWindow.X - 150
	marginY := gameWindow.Y + w.topOffset + 48

	fmt.Printf("marginX, marginY: %v %v\n", marginX, marginY)

	return marginX, marginY, nil
}
