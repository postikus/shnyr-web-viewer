package click_manager

import (
	"image"
	"syscall"
	"unsafe"

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

// KeyDown отправляет команду нажатия клавиши вниз
func (m *ClickManager) KeyDown(key string) {
	arduino.KeyDown(m.config, key)
}

// KeyUp отправляет команду отпускания клавиши
func (m *ClickManager) KeyUp(key string) {
	arduino.KeyUp(m.config, key)
}

// Paste выполняет вставку из буфера обмена (Ctrl+V)
func (m *ClickManager) Paste() {
	arduino.Paste(m.config)
}

// CopyToClipboard копирует текст в буфер обмена Windows
func (m *ClickManager) CopyToClipboard(text string) {
	// Windows API функции
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")

	openClipboard := user32.NewProc("OpenClipboard")
	emptyClipboard := user32.NewProc("EmptyClipboard")
	setClipboardData := user32.NewProc("SetClipboardData")
	closeClipboard := user32.NewProc("CloseClipboard")
	globalAlloc := kernel32.NewProc("GlobalAlloc")
	globalLock := kernel32.NewProc("GlobalLock")
	globalUnlock := kernel32.NewProc("GlobalUnlock")
	lstrcpy := kernel32.NewProc("lstrcpyW")

	// Константы
	const (
		CF_UNICODETEXT = 13
		GMEM_MOVEABLE  = 0x0002
	)

	// Открываем буфер обмена
	ret, _, _ := openClipboard.Call(0)
	if ret == 0 {
		return
	}
	defer closeClipboard.Call()

	// Очищаем буфер обмена
	emptyClipboard.Call()

	// Конвертируем строку в UTF-16
	textUTF16, err := syscall.UTF16FromString(text)
	if err != nil {
		return
	}

	// Выделяем память
	size := len(textUTF16)*2 + 2 // +2 для null-terminator
	hMem, _, _ := globalAlloc.Call(GMEM_MOVEABLE, uintptr(size))
	if hMem == 0 {
		return
	}

	// Блокируем память
	lpMem, _, _ := globalLock.Call(hMem)
	if lpMem == 0 {
		return
	}

	// Копируем данные
	lstrcpy.Call(lpMem, uintptr(unsafe.Pointer(&textUTF16[0])))

	// Разблокируем память
	globalUnlock.Call(hMem)

	// Устанавливаем данные в буфер обмена
	setClipboardData.Call(CF_UNICODETEXT, hMem)
}
