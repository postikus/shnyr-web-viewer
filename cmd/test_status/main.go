package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🧪 Тестирование системы статусов ШНЫРЯ")

	// Подключаемся к базе данных
	db, err := sql.Open("mysql", "root:tY6@uI!oP_aZ8$cV@tcp(108.181.194.102:3306)/octopus?parseTime=true")
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Проверяем подключение
	err = db.Ping()
	if err != nil {
		log.Fatalf("Ошибка проверки подключения: %v", err)
	}

	fmt.Println("✅ Подключение к базе данных установлено")

	// Функции для работы со статусом
	updateStatus := func(status string) error {
		_, err := db.Exec("INSERT INTO status (current_status) VALUES (?)", status)
		return err
	}

	addAction := func(action string) error {
		_, err := db.Exec("INSERT INTO actions (action) VALUES (?)", action)
		return err
	}

	// Тестируем различные статусы
	statuses := []string{"main", "ready", "cycle_all_items", "cycle_listed_items", "stopped"}

	for _, status := range statuses {
		fmt.Printf("📝 Обновляем статус на: %s\n", status)

		err = updateStatus(status)
		if err != nil {
			log.Printf("❌ Ошибка обновления статуса %s: %v", status, err)
			continue
		}

		err = addAction(fmt.Sprintf("Тест: статус изменен на %s", status))
		if err != nil {
			log.Printf("❌ Ошибка добавления действия: %v", err)
		}

		fmt.Printf("✅ Статус %s установлен\n", status)

		// Пауза между обновлениями
		time.Sleep(2 * time.Second)
	}

	fmt.Println("🎉 Тестирование завершено!")
	fmt.Println("🌐 Откройте веб-интерфейс http://localhost:8080 для просмотра статусов")
}
