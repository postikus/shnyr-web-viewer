package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: go run main.go <команда> [аргументы]")
		fmt.Println("Команды:")
		fmt.Println("  status <новый_статус> - обновить статус")
		fmt.Println("  action <действие> - добавить действие")
		fmt.Println("  show - показать текущий статус и последние действия")
		return
	}

	// Подключаемся к базе данных
	dsn := "root:tY6@uI!oP_aZ8$cV@tcp(108.181.194.102:3306)/octopus?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	command := os.Args[1]

	switch command {
	case "status":
		if len(os.Args) < 3 {
			fmt.Println("Ошибка: укажите новый статус")
			return
		}
		newStatus := os.Args[2]
		err := updateStatus(db, newStatus)
		if err != nil {
			log.Fatalf("Ошибка обновления статуса: %v", err)
		}
		fmt.Printf("Статус обновлен на: %s\n", newStatus)

	case "action":
		if len(os.Args) < 3 {
			fmt.Println("Ошибка: укажите действие")
			return
		}
		action := os.Args[2]
		err := addAction(db, action)
		if err != nil {
			log.Fatalf("Ошибка добавления действия: %v", err)
		}
		fmt.Printf("Действие добавлено: %s\n", action)

	case "show":
		status, actions, err := getStatusAndActions(db)
		if err != nil {
			log.Fatalf("Ошибка получения данных: %v", err)
		}
		fmt.Printf("Текущий статус: %s (обновлен: %s)\n", status.CurrentStatus, status.UpdatedAt)
		fmt.Println("Последние действия:")
		for _, action := range actions {
			fmt.Printf("  - %s (%s)\n", action.Action, action.CreatedAt)
		}

	default:
		fmt.Printf("Неизвестная команда: %s\n", command)
	}
}

func updateStatus(db *sql.DB, status string) error {
	_, err := db.Exec("INSERT INTO status (current_status) VALUES (?)", status)
	return err
}

func addAction(db *sql.DB, action string) error {
	_, err := db.Exec("INSERT INTO actions (action) VALUES (?)", action)
	return err
}

func getStatusAndActions(db *sql.DB) (Status, []Action, error) {
	var status Status
	err := db.QueryRow("SELECT id, current_status, updated_at FROM status ORDER BY id DESC LIMIT 1").Scan(&status.ID, &status.CurrentStatus, &status.UpdatedAt)
	if err != nil {
		return Status{}, nil, err
	}

	rows, err := db.Query("SELECT id, action, created_at FROM actions ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		return status, nil, err
	}
	defer rows.Close()

	var actions []Action
	for rows.Next() {
		var action Action
		err := rows.Scan(&action.ID, &action.Action, &action.CreatedAt)
		if err != nil {
			continue
		}
		actions = append(actions, action)
	}

	return status, actions, nil
}

type Status struct {
	ID            int
	CurrentStatus string
	UpdatedAt     string
}

type Action struct {
	ID        int
	Action    string
	CreatedAt string
}
