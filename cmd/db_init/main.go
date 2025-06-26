package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Подключаемся к MySQL без указания базы
	dsn := "root:root@tcp(108.181.194.102:3306)/"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к MySQL: %v", err)
	}
	defer db.Close()

	// Удаляем базу, если есть
	_, err = db.Exec("DROP DATABASE IF EXISTS octopus")
	if err != nil {
		log.Fatalf("Ошибка удаления базы: %v", err)
	}
	fmt.Println("База данных octopus удалена (если была)")

	// Создаём базу
	_, err = db.Exec("CREATE DATABASE octopus CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	if err != nil {
		log.Fatalf("Ошибка создания базы: %v", err)
	}
	fmt.Println("База данных octopus создана")

	// Подключаемся к новой базе
	db2, err := sql.Open("mysql", dsn+"octopus")
	if err != nil {
		log.Fatalf("Ошибка подключения к новой базе: %v", err)
	}
	defer db2.Close()

	// Создаём таблицу
	tableSQL := `CREATE TABLE ocr_results (
		id INT AUTO_INCREMENT PRIMARY KEY,
		image_path VARCHAR(255) NOT NULL,
		image_data LONGBLOB,
		ocr_text LONGTEXT,
		debug_info LONGTEXT,
		json_data LONGTEXT,
		raw_text LONGTEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err = db2.Exec(tableSQL)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы ocr_results: %v", err)
	}
	fmt.Println("Таблица ocr_results создана")

	// Создаём таблицу предметов с категориями
	itemsTableSQL := `CREATE TABLE items_list (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		category VARCHAR(50) NOT NULL DEFAULT 'consumables',
		min_price DECIMAL(15,2) DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err = db2.Exec(itemsTableSQL)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы items_list: %v", err)
	}
	fmt.Println("Таблица items_list создана")

	// Создаём таблицу для структурированных данных
	structuredTableSQL := `CREATE TABLE structured_items (
		id INT AUTO_INCREMENT PRIMARY KEY,
		ocr_result_id INT,
		title VARCHAR(255) NOT NULL,
		title_short VARCHAR(255),
		enhancement VARCHAR(10),
		price VARCHAR(50) NOT NULL,
		package BOOLEAN DEFAULT FALSE,
		owner VARCHAR(255),
		count VARCHAR(100),
		category VARCHAR(50),
		item_list_id INT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ocr_result_id) REFERENCES ocr_results(id) ON DELETE CASCADE,
		FOREIGN KEY (item_list_id) REFERENCES items_list(id) ON DELETE SET NULL
	)`
	_, err = db2.Exec(structuredTableSQL)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы structured_items: %v", err)
	}
	fmt.Println("Таблица structured_items создана")

	// Создаем таблицу для исправлений данных
	_, err = db2.Exec(`
		CREATE TABLE IF NOT EXISTS item_corrections (
			id INTEGER PRIMARY KEY AUTO_INCREMENT,
			item_id INTEGER NOT NULL,
			field_name VARCHAR(50) NOT NULL,
			current_value TEXT,
			corrected_value TEXT NOT NULL,
			comment TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (item_id) REFERENCES structured_items(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		log.Fatal("Ошибка создания таблицы item_corrections:", err)
	}

	fmt.Println("Инициализация базы завершена!")
}
