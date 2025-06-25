package image

import (
	"fmt"
	"image"
	"shnyr/internal/config"

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

// CheckButtonActive проверяет активность кнопки
func (h *ImageHelper) CheckButtonActive(buttonX, buttonY int, buttonName string, img image.Image) bool {
	buttonRPx, _, _, _ := GetPixelColor(img, buttonX, 36)
	fmt.Printf("%s RPx: %v\n", buttonName, buttonRPx)
	return buttonRPx == 86
}

// ButtonStatus содержит статус всех кнопок
type ButtonStatus struct {
	Button2Active bool
	Button3Active bool
	Button4Active bool
	Button5Active bool
	Button6Active bool
}

// CheckAllButtonsStatus проверяет статус всех кнопок на изображении
func CheckAllButtonsStatus(img image.Image, config *config.Config, marginX, marginY int) ButtonStatus {
	imageHelper := NewImageHelper(nil, config, marginX, marginY)

	button2Active := imageHelper.CheckButtonActive(config.Click.Button2.X, config.Click.Button2.Y, "listButton2", img)
	button3Active := imageHelper.CheckButtonActive(config.Click.Button3.X, config.Click.Button3.Y, "listButton3", img)
	button4Active := imageHelper.CheckButtonActive(config.Click.Button4.X, config.Click.Button4.Y, "listButton4", img)
	button5Active := imageHelper.CheckButtonActive(config.Click.Button5.X, config.Click.Button5.Y, "listButton5", img)
	button6Active := imageHelper.CheckButtonActive(config.Click.Button6.X, config.Click.Button6.Y, "listButton6", img)

	fmt.Printf("📋 Статус кнопок: Button2=%v, Button3=%v, Button4=%v, Button5=%v, Button6=%v\n",
		button2Active, button3Active, button4Active, button5Active, button6Active)

	return ButtonStatus{
		Button2Active: button2Active,
		Button3Active: button3Active,
		Button4Active: button4Active,
		Button5Active: button5Active,
		Button6Active: button6Active,
	}
}
