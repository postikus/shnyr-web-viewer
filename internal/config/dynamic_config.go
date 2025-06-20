package config

import (
	imageInternal "octopus/internal/image"
	"octopus/internal/screen"
	"octopus/internal/types"
)

// DynamicConfig расширяет Config возможностью динамического поиска окна
type DynamicConfig struct {
	*Config
	GameWindow *types.GameWindow
}

// NewDynamicConfig создает новый DynamicConfig
func NewDynamicConfig(config *Config) *DynamicConfig {
	return &DynamicConfig{
		Config: config,
	}
}

// FindAndSetGameWindow находит окно игры и устанавливает его координаты
func (dc *DynamicConfig) FindAndSetGameWindow() error {
	// Захватываем весь экран
	fullScreen, err := screen.CaptureFullScreen()
	if err != nil {
		return err
	}

	// Ищем окно игры
	gameWindow, err := imageInternal.FindGameWindow(fullScreen)
	if err != nil {
		return err
	}

	dc.GameWindow = gameWindow
	return nil
}

// GetAbsoluteCoordinates возвращает абсолютные координаты для клика
func (dc *DynamicConfig) GetAbsoluteCoordinates(relCoords Coordinates) (int, int) {
	if dc.GameWindow == nil {
		// Если окно не найдено, используем координаты как есть
		return relCoords.X, relCoords.Y
	}
	return imageInternal.ConvertToAbsoluteCoordinates(dc.GameWindow, relCoords.X, relCoords.Y)
}

// GetAbsoluteCoordinatesWithSize возвращает абсолютные координаты для скриншота
func (dc *DynamicConfig) GetAbsoluteCoordinatesWithSize(relCoords CoordinatesWithSize) CoordinatesWithSize {
	if dc.GameWindow == nil {
		// Если окно не найдено, используем координаты как есть
		return relCoords
	}

	absX, absY := imageInternal.ConvertToAbsoluteCoordinates(dc.GameWindow, relCoords.X, relCoords.Y)
	return CoordinatesWithSize{
		X:      absX,
		Y:      absY,
		Width:  relCoords.Width,
		Height: relCoords.Height,
	}
}
