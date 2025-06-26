package cycle_listed_items

import (
	"fmt"
	"shnyr/internal/database"
	"shnyr/internal/interrupt"
	"shnyr/internal/logger"
)

// checkForStopAction проверяет наличие действия "stop" в базе данных
func checkForStopAction(dbManager *database.DatabaseManager, loggerManager *logger.LoggerManager) (bool, int, error) {
	action, actionID, err := dbManager.GetLatestUnexecutedAction()
	if err != nil {
		return false, 0, err
	}

	if action == "stop" {
		loggerManager.Info("🛑 Обнаружено действие 'stop' в базе данных (ID: %d)", actionID)
		return true, actionID, nil
	}

	return false, 0, nil
}

// checkInterruption проверяет прерывание как через горячие клавиши, так и через базу данных
func checkInterruption(interruptManager *interrupt.InterruptManager, dbManager *database.DatabaseManager, loggerManager *logger.LoggerManager) (bool, int, error) {
	// Проверяем прерывание через горячие клавиши
	select {
	case <-interruptManager.GetScriptInterruptChan():
		loggerManager.Info("⏹️ Прерывание по горячим клавишам")
		return true, 0, fmt.Errorf("прерывание по горячим клавишам")
	default:
	}

	// Проверяем прерывание через базу данных
	hasStopAction, actionID, err := checkForStopAction(dbManager, loggerManager)
	if err != nil {
		loggerManager.LogError(err, "Ошибка проверки действий в базе данных")
		return false, 0, err
	}

	if hasStopAction {
		// Помечаем действие как выполненное
		err = dbManager.MarkActionAsExecuted(actionID)
		if err != nil {
			loggerManager.LogError(err, "Ошибка пометки действия как выполненного")
		}

		// Обновляем статус на stopped
		err = dbManager.UpdateStatus("stopped")
		if err != nil {
			loggerManager.LogError(err, "Ошибка обновления статуса на stopped")
		}

		loggerManager.Info("⏹️ Прерывание по действию 'stop' из базы данных")
		return true, actionID, fmt.Errorf("прерывание по действию 'stop' из базы данных")
	}

	return false, 0, nil
}
