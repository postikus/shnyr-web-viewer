package click_manager

import (
	"image"

	"github.com/tarm/serial"

	"shnyr/internal/arduino"
	"shnyr/internal/config"
	"shnyr/internal/database"
	"shnyr/internal/logger"
)

// ScreenshotManager интерфейс для работы со скриншотами
type ScreenshotManager interface {
	CaptureScreenShot() (image.Image, error)
	CheckScrollExists() bool
}

// ClickManager управляет кликами и скроллом
type ClickManager struct {
	port              *serial.Port
	config            *config.Config
	marginX           int
	marginY           int
	screenshotManager ScreenshotManager
	dbManager         *database.DatabaseManager
	logger            *logger.LoggerManager
}

// NewClickManager создает новый экземпляр ClickManager
func NewClickManager(port *serial.Port, config *config.Config, marginX, marginY int, screenshotManager ScreenshotManager, dbManager *database.DatabaseManager, loggerManager *logger.LoggerManager) *ClickManager {
	return &ClickManager{
		port:              port,
		config:            config,
		marginX:           marginX,
		marginY:           marginY,
		screenshotManager: screenshotManager,
		dbManager:         dbManager,
		logger:            loggerManager,
	}
}

// FocusL2Window фокусирует окно L2, кликая по координатам Item1
func (m *ClickManager) FocusL2Window() {
	finalCoordinates := image.Point{
		X: 30,
		Y: 30,
	}
	arduino.ClickCoordinates(m.config, finalCoordinates)
}

// ClickCoordinates выполняет клик по указанным координатам с учетом отступов
func (m *ClickManager) ClickCoordinates(coordinate image.Point) {
	finalCoordinates := image.Point{
		X: m.marginX + coordinate.X,
		Y: m.marginY + coordinate.Y,
	}
	arduino.ClickCoordinates(m.config, finalCoordinates)
}
