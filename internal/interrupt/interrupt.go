package interrupt

import (
	"shnyr/internal/logger"

	"github.com/moutend/go-hook/pkg/keyboard"
	"github.com/moutend/go-hook/pkg/types"
)

// InterruptManager управляет прерываниями и горячими клавишами
type InterruptManager struct {
	scriptInterruptChan chan bool
	scriptStartChan     chan bool
	isScriptRunning     *bool
	isInterrupted       *bool
	loggerManager       *logger.LoggerManager
	lastScriptType      string
}

// NewInterruptManager создает новый менеджер прерываний
func NewInterruptManager(loggerManager *logger.LoggerManager) *InterruptManager {
	isScriptRunning := false
	isInterrupted := false
	return &InterruptManager{
		scriptInterruptChan: make(chan bool, 1),
		scriptStartChan:     make(chan bool, 1),
		isScriptRunning:     &isScriptRunning,
		isInterrupted:       &isInterrupted,
		loggerManager:       loggerManager,
		lastScriptType:      "",
	}
}

// StartMonitoring запускает мониторинг горячих клавиш
func (im *InterruptManager) StartMonitoring() {
	go im.monitorHotkeys()
}

// GetScriptInterruptChan возвращает канал для прерывания скрипта
func (im *InterruptManager) GetScriptInterruptChan() <-chan bool {
	return im.scriptInterruptChan
}

// GetScriptStartChan возвращает канал для запуска скрипта
func (im *InterruptManager) GetScriptStartChan() <-chan bool {
	return im.scriptStartChan
}

// SetScriptRunning устанавливает состояние выполнения скрипта
func (im *InterruptManager) SetScriptRunning(running bool) {
	*im.isScriptRunning = running
}

// IsScriptRunning возвращает состояние выполнения скрипта
func (im *InterruptManager) IsScriptRunning() bool {
	return *im.isScriptRunning
}

// GetLastScriptType возвращает тип последнего запущенного скрипта
func (im *InterruptManager) GetLastScriptType() string {
	return im.lastScriptType
}

// SetLastScriptType устанавливает тип последнего запущенного скрипта
func (im *InterruptManager) SetLastScriptType(scriptType string) {
	im.lastScriptType = scriptType
}

// IsInterrupted возвращает состояние прерывания
func (im *InterruptManager) IsInterrupted() bool {
	return *im.isInterrupted
}

// SetInterrupted устанавливает состояние прерывания
func (im *InterruptManager) SetInterrupted(interrupted bool) {
	*im.isInterrupted = interrupted
}

// monitorHotkeys мониторит горячие клавиши
func (im *InterruptManager) monitorHotkeys() {
	eventChan := make(chan types.KeyboardEvent, 100)
	go keyboard.Install(nil, eventChan)
	defer keyboard.Uninstall()

	shiftPressed := false
	ctrlPressed := false

	for event := range eventChan {
		// Отслеживаем Shift
		if event.Message == types.WM_KEYDOWN && (event.VKCode == types.VK_LSHIFT || event.VKCode == types.VK_RSHIFT) {
			shiftPressed = true
		}
		if event.Message == types.WM_KEYUP && (event.VKCode == types.VK_LSHIFT || event.VKCode == types.VK_RSHIFT) {
			shiftPressed = false
		}

		// Отслеживаем Ctrl
		if event.Message == types.WM_KEYDOWN && (event.VKCode == types.VK_LCONTROL || event.VKCode == types.VK_RCONTROL) {
			ctrlPressed = true
		}
		if event.Message == types.WM_KEYUP && (event.VKCode == types.VK_LCONTROL || event.VKCode == types.VK_RCONTROL) {
			ctrlPressed = false
		}

		// Ctrl+Shift+1 для cycle_all_items
		if event.Message == types.WM_KEYDOWN && event.VKCode == types.VK_1 && shiftPressed && ctrlPressed {
			im.lastScriptType = "cycle_all_items"
			im.scriptStartChan <- true
		}

		// Ctrl+Shift+2 для cycle_listed_items
		if event.Message == types.WM_KEYDOWN && event.VKCode == types.VK_2 && shiftPressed && ctrlPressed {
			im.lastScriptType = "cycle_listed_items"
			im.scriptStartChan <- true
		}

		// Q для прерывания
		if event.Message == types.WM_KEYDOWN && (event.VKCode == types.VK_Q || event.VKCode == types.VK_CAPITAL) {
			// Q всегда только прерывает скрипт, если он запущен
			if im.isScriptRunning != nil && *im.isScriptRunning {
				*im.isInterrupted = true
				im.scriptInterruptChan <- true
			}
		}
	}
}
