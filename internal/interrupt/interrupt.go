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
	loggerManager       *logger.LoggerManager
}

// NewInterruptManager создает новый менеджер прерываний
func NewInterruptManager(loggerManager *logger.LoggerManager) *InterruptManager {
	isScriptRunning := false
	return &InterruptManager{
		scriptInterruptChan: make(chan bool, 1),
		scriptStartChan:     make(chan bool, 1),
		isScriptRunning:     &isScriptRunning,
		loggerManager:       loggerManager,
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

// monitorHotkeys мониторит горячие клавиши
func (im *InterruptManager) monitorHotkeys() {
	eventChan := make(chan types.KeyboardEvent, 100)
	go keyboard.Install(nil, eventChan)
	defer keyboard.Uninstall()

	shiftPressed := false

	for event := range eventChan {
		if event.Message == types.WM_KEYDOWN && (event.VKCode == types.VK_LSHIFT || event.VKCode == types.VK_RSHIFT) {
			shiftPressed = true
		}
		if event.Message == types.WM_KEYUP && (event.VKCode == types.VK_LSHIFT || event.VKCode == types.VK_RSHIFT) {
			shiftPressed = false
		}
		if event.Message == types.WM_KEYDOWN && event.VKCode == types.VK_RETURN && shiftPressed {
			im.scriptStartChan <- true
		}
		if event.Message == types.WM_KEYDOWN && (event.VKCode == types.VK_Q || event.VKCode == types.VK_CAPITAL) {
			// Q всегда только прерывает script1, если он запущен
			if im.isScriptRunning != nil && *im.isScriptRunning {
				im.scriptInterruptChan <- true
			}
		}
	}
}

// LogInstructions выводит инструкции по использованию горячих клавиш
func (im *InterruptManager) LogInstructions() {
	im.loggerManager.Info("⏸️ Программа готова к работе. Нажмите Shift+Enter для запуска script1, Q для прерывания")
	im.loggerManager.Info("🔥 Горячие клавиши: Shift+Enter для запуска, Q для прерывания script1")
}
