package helpers

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"octopus/internal/config"
	imageInternal "octopus/internal/image"
	"octopus/internal/scripts"

	"github.com/tarm/serial"
)

// ImageHelper содержит функции для работы с изображениями
type ImageHelper struct {
	port    *serial.Port
	config  *config.Config
	marginX int
	marginY int
}

// NewImageHelper создает новый экземпляр ImageHelper
func NewImageHelper(port *serial.Port, config *config.Config, marginX, marginY int) *ImageHelper {
	return &ImageHelper{
		port:    port,
		config:  config,
		marginX: marginX,
		marginY: marginY,
	}
}

// CheckAndScreenScroll проверяет и выполняет скролл экрана
func (h *ImageHelper) CheckAndScreenScroll(counter int, x int, img image.Image) (int, int) {
	r, _, _, _ := imageInternal.GetPixelColor(img, 297, 320)
	fmt.Printf("r: %v\n", r)
	if r < 50 {
		scripts.ScrollDown(h.port, h.config, x)
	}
	return counter + 1, r
}

// CheckAndClickScreenScroll проверяет и выполняет клик для скролла
func (h *ImageHelper) CheckAndClickScreenScroll(counter int, img image.Image) (int, int) {
	r, _, _, _ := imageInternal.GetPixelColor(img, 297, 342)
	if r < 50 {
		scripts.FastClick(h.port, h.config)
	}
	return counter + 1, r
}

// CheckButtonActive проверяет активность кнопки
func (h *ImageHelper) CheckButtonActive(buttonX, buttonY int, buttonName string, img image.Image) bool {
	buttonRPx, _, _, _ := imageInternal.GetPixelColor(img, buttonX, 36)
	fmt.Printf("%s RPx: %v\n", buttonName, buttonRPx)
	return buttonRPx == 86
}

// ImageToBytes конвертирует изображение в байты в формате PNG
func ImageToBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("ошибка кодирования изображения: %v", err)
	}
	return buf.Bytes(), nil
}

// Min возвращает минимальное из двух чисел
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
