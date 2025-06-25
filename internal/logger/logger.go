package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// LogLevel представляет уровень логирования
type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	ERROR LogLevel = "ERROR"
)

// LoggerManager управляет логированием в файл
type LoggerManager struct {
	file   *os.File
	logger *log.Logger
}

// NewLoggerManager создает новый экземпляр LoggerManager
func NewLoggerManager(logFilePath string) (*LoggerManager, error) {
	// Создаем директорию для логов, если её нет
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("ошибка создания директории для логов: %v", err)
	}

	// Открываем файл для записи (создаем, если не существует)
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла логов: %v", err)
	}

	// Создаем логгер с префиксом и флагами
	logger := log.New(file, "", log.LstdFlags)

	return &LoggerManager{
		file:   file,
		logger: logger,
	}, nil
}

// Close закрывает файл логов
func (l *LoggerManager) Close() error {
	return l.file.Close()
}

// logWithLevel записывает сообщение с указанным уровнем
func (l *LoggerManager) logWithLevel(level LogLevel, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] %s: %s", timestamp, level, message)

	// Записываем в файл
	l.logger.Println(logEntry)

	// Также выводим в консоль для удобства отладки
	fmt.Println(logEntry)
}

// Debug записывает отладочное сообщение
func (l *LoggerManager) Debug(format string, args ...interface{}) {
	l.logWithLevel(DEBUG, format, args...)
}

// Info записывает информационное сообщение
func (l *LoggerManager) Info(format string, args ...interface{}) {
	l.logWithLevel(INFO, format, args...)
}

// Error записывает сообщение об ошибке
func (l *LoggerManager) Error(format string, args ...interface{}) {
	l.logWithLevel(ERROR, format, args...)
}

// LogError записывает ошибку с дополнительной информацией
func (l *LoggerManager) LogError(err error, context string) {
	if err != nil {
		l.Error("%s: %v", context, err)
	}
}
