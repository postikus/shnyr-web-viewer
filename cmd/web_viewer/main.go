package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"encoding/base64"

	_ "github.com/go-sql-driver/mysql"
)

type StructuredItem struct {
	ID          int
	OCRResultID int
	Title       string
	TitleShort  string
	Enhancement string
	Price       string
	Package     bool
	Owner       string
	CreatedAt   string
}

type OCRResult struct {
	ID        int
	ImagePath string
	ImageData []byte
	OCRText   string
	DebugInfo string
	JSONData  string
	RawText   string
	CreatedAt string
	Items     []StructuredItem
}

type PageData struct {
	Results     []OCRResult
	CurrentPage int
	TotalPages  int
	TotalCount  int
	HasPrev     bool
	HasNext     bool
	PrevPage    int
	NextPage    int
	SearchQuery string
	MinPrice    string
	MaxPrice    string
}

func getDatabaseDSN() string {
	// Получаем параметры подключения из переменных окружения
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "108.181.194.102"
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "root"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "octopus"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbPort, dbName)
}

func main() {
	// Получаем порт из переменной окружения
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Подключаемся к базе данных
	dbDSN := getDatabaseDSN()
	db, err := sql.Open("mysql", dbDSN)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка проверки подключения к базе данных: %v", err)
	}

	log.Printf("Успешно подключились к базе данных: %s", dbDSN)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Получаем параметры пагинации и поиска
		pageStr := r.URL.Query().Get("page")
		searchQuery := r.URL.Query().Get("search")
		minPrice := r.URL.Query().Get("min_price")
		maxPrice := r.URL.Query().Get("max_price")
		page := 1
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		resultsPerPage := 10
		offset := (page - 1) * resultsPerPage

		// Формируем SQL запрос с поиском
		var countQuery, dataQuery string
		var args []interface{}

		if searchQuery != "" || minPrice != "" || maxPrice != "" {
			// Поиск по структурированным данным
			countQuery = `SELECT COUNT(DISTINCT ocr.id) FROM ocr_results ocr 
				LEFT JOIN structured_items si ON ocr.id = si.ocr_result_id 
				WHERE (si.title LIKE ? OR si.owner LIKE ? OR si.price LIKE ? OR si.title_short LIKE ?)`
			dataQuery = `SELECT DISTINCT ocr.id, ocr.image_path, ocr.image_data, ocr.ocr_text, ocr.debug_info, ocr.json_data, ocr.raw_text, ocr.created_at 
				FROM ocr_results ocr 
				LEFT JOIN structured_items si ON ocr.id = si.ocr_result_id 
				WHERE (si.title LIKE ? OR si.owner LIKE ? OR si.price LIKE ? OR si.title_short LIKE ?)`

			searchPattern := "%" + searchQuery + "%"
			args = []interface{}{searchPattern, searchPattern, searchPattern, searchPattern}

			// Добавляем фильтрацию по цене
			if minPrice != "" || maxPrice != "" {
				countQuery += ` AND (`
				dataQuery += ` AND (`
				priceConditions := []string{}
				priceArgs := []interface{}{}

				if minPrice != "" {
					priceConditions = append(priceConditions, "CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(10,2)) >= ?")
					priceArgs = append(priceArgs, minPrice)
				}

				if maxPrice != "" {
					priceConditions = append(priceConditions, "CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(10,2)) <= ?")
					priceArgs = append(priceArgs, maxPrice)
				}

				countQuery += strings.Join(priceConditions, " AND ") + ")"
				dataQuery += strings.Join(priceConditions, " AND ") + ")"
				args = append(args, priceArgs...)
			}

			dataQuery += ` ORDER BY ocr.created_at DESC LIMIT ? OFFSET ?`
		} else {
			// Без поиска
			countQuery = "SELECT COUNT(*) FROM ocr_results"
			dataQuery = `SELECT id, image_path, image_data, ocr_text, debug_info, json_data, raw_text, created_at FROM ocr_results ORDER BY created_at DESC LIMIT ? OFFSET ?`
		}

		// Получаем общее количество записей
		var totalCount int
		var countArgs []interface{}
		if searchQuery != "" || minPrice != "" || maxPrice != "" {
			countArgs = args
		}
		err := db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
		if err != nil {
			http.Error(w, "DB error", 500)
			return
		}

		// Вычисляем общее количество страниц
		totalPages := (totalCount + resultsPerPage - 1) / resultsPerPage
		if totalPages == 0 {
			totalPages = 1
		}

		// Проверяем, что текущая страница не превышает общее количество
		if page > totalPages {
			page = totalPages
			offset = (page - 1) * resultsPerPage
		}

		// Получаем записи для текущей страницы
		var rows *sql.Rows
		if searchQuery != "" || minPrice != "" || maxPrice != "" {
			args = append(args, resultsPerPage, offset)
			rows, err = db.Query(dataQuery, args...)
		} else {
			rows, err = db.Query(dataQuery, resultsPerPage, offset)
		}

		if err != nil {
			http.Error(w, "DB error", 500)
			return
		}
		defer rows.Close()

		var results []OCRResult
		for rows.Next() {
			var res OCRResult
			if err := rows.Scan(&res.ID, &res.ImagePath, &res.ImageData, &res.OCRText, &res.DebugInfo, &res.JSONData, &res.RawText, &res.CreatedAt); err != nil {
				continue
			}

			// Загружаем структурированные данные для этого OCR результата
			itemRows, err := db.Query(`SELECT id, ocr_result_id, title, title_short, enhancement, price, package, owner, created_at FROM structured_items WHERE ocr_result_id = ? ORDER BY created_at`, res.ID)
			if err == nil {
				defer itemRows.Close()
				for itemRows.Next() {
					var item StructuredItem
					if err := itemRows.Scan(&item.ID, &item.OCRResultID, &item.Title, &item.TitleShort, &item.Enhancement, &item.Price, &item.Package, &item.Owner, &item.CreatedAt); err == nil {
						res.Items = append(res.Items, item)
					}
				}
			}

			results = append(results, res)
		}

		// Подготавливаем данные для шаблона
		pageData := PageData{
			Results:     results,
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalCount:  totalCount,
			HasPrev:     page > 1,
			HasNext:     page < totalPages,
			PrevPage:    page - 1,
			NextPage:    page + 1,
			SearchQuery: searchQuery,
			MinPrice:    minPrice,
			MaxPrice:    maxPrice,
		}

		tmpl := `
		<html>
		<head>
			<title>OCR Results</title>
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>
				* { box-sizing: border-box; }
				body {
					font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
					margin: 0;
					padding: 0;
					background: #fff;
					min-height: 100vh;
					color: #111;
				}
				.container {
					width: 100%;
					margin: 0;
					background: #fff;
					border-radius: 0;
					box-shadow: none;
					overflow: hidden;
				}
				.header {
					position: fixed;
					top: 0;
					left: 0;
					right: 0;
					background: #fff;
					color: #111;
					padding: 20px 0;
					z-index: 1000;
					border-bottom: 2px solid #eee;
				}
				.header-content {
					display: flex;
					justify-content: space-between;
					align-items: center;
					padding: 0 30px;
					flex-wrap: wrap;
					gap: 20px;
				}
				.title-section h1 {
					margin: 0;
					font-size: 2.5em;
					font-weight: 700;
					color: #111;
					text-shadow: none;
				}
				.search-container {
					display: flex;
					align-items: center;
					gap: 15px;
					flex-wrap: wrap;
				}
				.search-input {
					padding: 12px 20px;
					border: 1px solid #bbb;
					border-radius: 25px;
					font-size: 16px;
					width: 300px;
					background: #fff;
					color: #111;
					box-shadow: none;
					transition: border-color 0.3s ease;
				}
				.search-input:focus {
					outline: none;
					border-color: #111;
				}
				.price-filter {
					display: flex;
					align-items: center;
					gap: 10px;
					background: #f5f5f5;
					padding: 8px 15px;
					border-radius: 20px;
				}
				.price-input {
					padding: 8px 12px;
					border: 1px solid #bbb;
					border-radius: 15px;
					width: 100px;
					font-size: 14px;
					background: #fff;
					color: #111;
				}
				.search-button {
					padding: 12px 20px;
					background: #111;
					color: #fff;
					border: none;
					border-radius: 25px;
					font-size: 16px;
					cursor: pointer;
					transition: background 0.3s ease;
				}
				.search-button:hover {
					background: #444;
				}
				.clear-button {
					padding: 8px 16px;
					background: #fff;
					color: #111;
					border: 1px solid #bbb;
					text-decoration: none;
					border-radius: 20px;
					font-size: 14px;
					transition: background 0.3s ease, color 0.3s ease;
				}
				.clear-button:hover {
					background: #eee;
					color: #000;
				}
				.tabs {
					display: flex;
					background: #f5f5f5;
					border-bottom: 2px solid #eee;
				}
				.tab {
					padding: 15px 30px;
					text-decoration: none;
					color: #444;
					font-weight: 600;
					transition: background 0.3s ease, color 0.3s ease;
					border-bottom: 3px solid transparent;
				}
				.tab:hover {
					background: #eee;
					color: #000;
				}
				.tab.active {
					color: #111;
					border-bottom-color: #111;
					background: #fff;
				}
				.content {
					padding: 30px;
					margin-top: 120px; /* Отступ для фиксированной шапки */
					margin-bottom: 100px; /* Отступ для фиксированной пагинации */
				}
				table {
					width: 100%;
					border-collapse: collapse;
					background: #fff;
					border-radius: 10px;
					overflow: hidden;
					box-shadow: none;
				}
				th, td {
					padding: 15px;
					text-align: left;
					border-bottom: 1px solid #eee;
				}
				th {
					background: #f5f5f5;
					color: #111;
					font-weight: 600;
					font-size: 14px;
				}
				tr:hover {
					background: #f0f0f0;
				}
				.screenshot-cell {
					width: 40px;
				}
				.screenshot-thumb {
					width: 33px;
					height: 20px;
					object-fit: cover;
					border-radius: 8px;
					cursor: pointer;
					transition: transform 0.3s ease;
					filter: grayscale(1);
				}
				.screenshot-thumb:hover {
					transform: scale(1.1);
				}
				.structured-data {
					max-width: 400px;
				}
				.structured-table {
					width: 100%;
					font-size: 12px;
					border-collapse: collapse;
				}
				.structured-table th,
				.structured-table td {
					padding: 4px 8px;
					border: 1px solid #ddd;
					font-size: 11px;
					background: #fff;
					color: #111;
				}
				.structured-table th {
					background: #f5f5f5;
					font-weight: 600;
				}
				.cheapest {
					background: #eaeaea !important;
					color: #111 !important;
					font-weight: bold !important;
				}
				.cheapest-package {
					background: #d4edda !important;
					color: #111 !important;
					font-weight: bold !important;
				}
				.structured-table tr.cheapest {
					background: #eaeaea !important;
					color: #111 !important;
					font-weight: bold !important;
				}
				.structured-table tr.cheapest-package {
					background: #d4edda !important;
					color: #111 !important;
					font-weight: bold !important;
				}
				.pagination {
					position: fixed;
					bottom: 0;
					left: 0;
					right: 0;
					display: flex;
					justify-content: center;
					align-items: center;
					margin-top: 30px;
					gap: 10px;
					background: #fff;
					padding: 15px;
					border-top: 2px solid #eee;
					z-index: 1000;
					flex-wrap: wrap;
				}
				.pagination a {
					padding: 10px 15px;
					background: #fff;
					color: #111;
					text-decoration: none;
					border-radius: 8px;
					border: 1px solid #bbb;
					transition: background 0.3s ease, color 0.3s ease;
				}
				.pagination a:hover {
					background: #eee;
					color: #000;
				}
				.pagination .current {
					background: #111;
					color: #fff;
					font-weight: bold;
				}
				.pagination .disabled {
					background: #eee;
					color: #bbb;
					cursor: not-allowed;
				}
				.pagination .disabled:hover {
					transform: none;
				}
				.modal {
					display: none;
					position: fixed;
					z-index: 2000;
					left: 0;
					top: 0;
					width: 100%;
					height: 100%;
					background-color: rgba(0,0,0,0.8);
					backdrop-filter: blur(5px);
				}
				.modal-content {
					position: relative;
					background-color: #fff;
					margin: 2% auto;
					padding: 20px;
					border-radius: 15px;
					width: 90%;
					max-width: 1200px;
					max-height: 90vh;
					overflow-y: auto;
					animation: modalSlideIn 0.3s ease;
					color: #111;
				}
				@keyframes modalSlideIn {
					from { opacity: 0; transform: translateY(-50px); }
					to { opacity: 1; transform: translateY(0); }
				}
				.close {
					position: absolute;
					right: 20px;
					top: 20px;
					font-size: 28px;
					font-weight: bold;
					cursor: pointer;
					color: #888;
					transition: color 0.3s ease;
				}
				.close:hover {
					color: #111;
				}
				.modal-image {
					max-width: 100%;
					height: auto;
					border-radius: 10px;
					box-shadow: none;
				}
				.modal-title {
					font-size: 24px;
					font-weight: bold;
					margin-bottom: 20px;
					color: #111;
				}
				.modal-section {
					margin: 20px 0;
					padding: 20px;
					background: #f5f5f5;
					border-radius: 10px;
				}
				.modal-section h3 {
					margin-top: 0;
					color: #111;
				}
				.detail-row {
					display: flex;
					align-items: center;
					gap: 10px;
					margin: 5px 0;
				}
				.detail-label {
					font-weight: 600;
					min-width: 120px;
				}
				.detail-value {
					flex: 1;
				}
				.correction-button {
					padding: 4px 8px;
					background: #fff;
					color: #111;
					border: 1px solid #bbb;
					border-radius: 4px;
					font-size: 10px;
					cursor: pointer;
					margin-left: 10px;
					transition: background 0.3s ease, color 0.3s ease;
				}
				.correction-button:hover {
					background: #eee;
					color: #000;
				}
				.correction-modal {
					display: none;
					position: fixed;
					z-index: 3000;
					left: 0;
					top: 0;
					width: 100%;
					height: 100%;
					background-color: rgba(0,0,0,0.9);
				}
				.correction-content {
					background-color: #fff;
					margin: 5% auto;
					padding: 30px;
					border-radius: 15px;
					width: 90%;
					max-width: 600px;
					animation: modalSlideIn 0.3s ease;
					color: #111;
				}
				.form-group {
					margin-bottom: 20px;
				}
				.form-group label {
					font-weight: 600;
					color: #111;
					font-size: 0.95em;
				}
				.correction-select,
				.correction-input,
				.correction-textarea {
					padding: 12px 16px;
					border: 1px solid #bbb;
					border-radius: 8px;
					font-size: 14px;
					background: #fff;
					color: #111;
					transition: border-color 0.3s ease;
				}
				.correction-select:focus,
				.correction-input:focus,
				.correction-textarea:focus {
					border-color: #111;
					outline: none;
				}
				.correction-input[readonly] {
					background-color: #f5f5f5;
					color: #888;
				}
				.correction-textarea {
					resize: vertical;
					min-height: 80px;
					font-family: inherit;
				}
				.correction-buttons {
					display: flex;
					gap: 15px;
					justify-content: center;
					margin-top: 20px;
				}
				.save-button,
				.cancel-button {
					padding: 12px 24px;
					border: none;
					border-radius: 8px;
					font-size: 14px;
					font-weight: 600;
					cursor: pointer;
					transition: background 0.3s ease, color 0.3s ease;
					min-width: 140px;
				}
				.save-button {
					background: #111;
					color: #fff;
				}
				.save-button:hover {
					background: #444;
				}
				.cancel-button {
					background: #fff;
					color: #111;
					border: 1px solid #bbb;
				}
				.cancel-button:hover {
					background: #eee;
					color: #000;
				}
				/* Мобильные устройства */
				@media (max-width: 768px) {
					.header {
						position: fixed;
						top: 0;
						left: 0;
						right: 0;
						padding: 10px 0;
						z-index: 1000;
					}
					
					.header-content {
						flex-direction: column;
						padding: 0 15px;
						gap: 10px;
					}
					
					.title-section h1 {
						font-size: 1.8em;
						text-align: center;
					}
					
					.search-container {
						flex-direction: column;
						width: 100%;
						gap: 10px;
					}
					
					.search-input {
						width: 100%;
						font-size: 14px;
					}
					
					.price-filter {
						width: 100%;
						justify-content: space-between;
					}
					
					.price-input {
						width: 80px;
						font-size: 12px;
					}
					
					.content {
						margin-top: 250px;
						padding: 15px;
						overflow-x: auto;
					}
					
					table {
						min-width: 800px;
						font-size: 12px;
					}
					
					th, td {
						padding: 8px 4px;
						font-size: 11px;
					}
					
					.screenshot-cell {
						width: 40px;
					}
					
					.screenshot-thumb {
						width: 33px;
						height: 20px;
					}
					
					.structured-data {
						max-width: 300px;
						overflow-x: auto;
					}
					
					.structured-table {
						min-width: 600px;
						font-size: 10px;
					}
					
					.structured-table th,
					.structured-table td {
						padding: 2px 4px;
						font-size: 9px;
					}
					
					.modal-content {
						width: 95%;
						margin: 5% auto;
						padding: 15px;
						max-height: 95vh;
					}
					
					.modal-section {
						padding: 15px;
						margin: 15px 0;
					}
					
					.pagination {
						flex-wrap: wrap;
						gap: 5px;
					}
					
					.pagination a,
					.pagination span {
						padding: 8px 12px;
						font-size: 12px;
					}
				}
				
				/* Очень маленькие экраны */
				@media (max-width: 480px) {
					.title-section h1 {
						font-size: 1.5em;
					}
					
					.content {
						margin-top: 230px;
						padding: 10px;
					}
					
					table {
						min-width: 700px;
					}
					
					.structured-table {
						min-width: 500px;
					}
				}
				.ocr-image {
					max-width: 220px;
					width: 100%;
					height: auto;
					display: block;
					margin: 0 auto;
				}
				@media (max-width: 600px) {
					.ocr-image {
						max-width: 45px;
					}
					.content {
						margin-top: 170px;
						margin-bottom: 120px;
						padding: 15px;
					}
					
					.pagination {
						padding: 10px;
						gap: 5px;
					}
					
					.pagination a,
					.pagination span {
						padding: 6px 8px;
						font-size: 11px;
						min-width: auto;
					}
				}
				
				/* Очень маленькие экраны */
				@media (max-width: 480px) {
					.content {
						margin-top: 230px;
						margin-bottom: 140px;
						padding: 10px;
					}
					
					.pagination {
						padding: 8px;
						gap: 3px;
					}
					
					.pagination a,
					.pagination span {
						padding: 5px 6px;
						font-size: 10px;
					}
				}
			</style>
		</head>
		<body>
		<div class="container">
			<div class="header">
				<h2 class="header-title">👓 ШНЫРЬ v0.1</h2>
				<div class="header-controls">
					<form method="GET" action="/" class="search-container">
						<input type="text" name="search" value="{{.SearchQuery}}" placeholder="Поиск по названию, владельцу, цене..." class="search-input">
						<div class="filter-row">
							<input type="number" name="min_price" value="{{.MinPrice}}" placeholder="Мин. цена" class="price-input" min="0" step="0.01">
							<input type="number" name="max_price" value="{{.MaxPrice}}" placeholder="Макс. цена" class="price-input" min="0" step="0.01">
							<button type="submit" class="search-button">🔍 Поиск</button>
							{{if or .SearchQuery .MinPrice .MaxPrice}}
							<a href="/" class="clear-button">❌ Очистить</a>
							{{end}}
						</div>
					</form>
				</div>
			</div>
			
			<div class="content">
				{{if .SearchQuery}}
				<div class="search-info">
					🔍 Поиск: "{{.SearchQuery}}" | Найдено: {{.TotalCount}} записей
				</div>
				{{end}}
				
				<div class="stats">
					📊 Страница {{.CurrentPage}} из {{.TotalPages}} | 
					📋 Показано {{len .Results}} записей из {{.TotalCount}}
				</div>

				<div class="mobile-table">
				<table>
				<tr>
					<th>Structured Data</th>
					<th>Screenshot</th>
					<th>Created</th>
				</tr>
				{{range .Results}}
				<tr data-raw-text="{{jsEscape .RawText}}" data-id="{{.ID}}" data-image="{{base64encode .ImageData}}" data-debug="{{jsEscape .DebugInfo}}" data-items="{{if .Items}}true{{else}}false{{end}}" data-structured-items='{{if .Items}}[{{range $index, $item := .Items}}{{if $index}},{{end}}{"title":"{{jsEscape $item.Title}}","titleShort":"{{jsEscape $item.TitleShort}}","enhancement":"{{jsEscape $item.Enhancement}}","price":"{{jsEscape $item.Price}}","package":{{$item.Package}},"owner":"{{jsEscape $item.Owner}}"}{{end}}]{{else}}[]{{end}}' onclick="openDetailModalFromData(this)" style="cursor: pointer;">
				<td>
					{{if .Items}}
					<div class="structured-table">
					<table>
					<tr><th>Title</th><th>Title Short</th><th>Enhancement</th><th>Price</th><th>Package</th><th>Owner</th></tr>
					{{range .Items}}
					<tr class="cheapest-item-{{.Enhancement}}-{{.Price}}">
					<td>{{.Title}}</td>
					<td>{{.TitleShort}}</td>
					<td>{{.Enhancement}}</td>
					<td>{{formatPrice .Price}}</td>
					<td>{{if .Package}}✔️{{end}}</td>
					<td>{{.Owner}}</td>
					</tr>
					{{end}}
					</table>
					</div>
					{{else}}
					<div class="no-data">No structured data</div>
					{{end}}
				</td>
				<td class="image-cell">
					{{if .ImageData}}
					<img src="data:image/png;base64,{{base64encode .ImageData}}" class="ocr-image" style="cursor: pointer;" />
					{{else}}
					<div class="no-data">No image data</div>
					{{end}}
				</td>
				<td class="date-cell">{{.CreatedAt}}</td>
				</tr>
				{{end}}
				</table>
				</div>

				<div class="pagination">
					{{if .HasPrev}}
						<a href="?page=1{{if .SearchQuery}}&search={{.SearchQuery}}{{end}}{{if .MinPrice}}&min_price={{.MinPrice}}{{end}}{{if .MaxPrice}}&max_price={{.MaxPrice}}{{end}}">« Первая</a>
						<a href="?page={{.PrevPage}}{{if .SearchQuery}}&search={{.SearchQuery}}{{end}}{{if .MinPrice}}&min_price={{.MinPrice}}{{end}}{{if .MaxPrice}}&max_price={{.MaxPrice}}{{end}}">‹ Предыдущая</a>
					{{else}}
						<span class="disabled">« Первая</span>
						<span class="disabled">‹ Предыдущая</span>
					{{end}}
					
					{{range $i := sequence .CurrentPage .TotalPages}}
						{{if eq $i $.CurrentPage}}
							<span class="current">{{$i}}</span>
						{{else}}
							<a href="?page={{$i}}{{if $.SearchQuery}}&search={{$.SearchQuery}}{{end}}{{if $.MinPrice}}&min_price={{$.MinPrice}}{{end}}{{if $.MaxPrice}}&max_price={{$.MaxPrice}}{{end}}">{{$i}}</a>
						{{end}}
					{{end}}
					
					{{if .HasNext}}
						<a href="?page={{.NextPage}}{{if .SearchQuery}}&search={{.SearchQuery}}{{end}}{{if .MinPrice}}&min_price={{.MinPrice}}{{end}}{{if .MaxPrice}}&max_price={{.MaxPrice}}{{end}}">Следующая ›</a>
						<a href="?page={{.TotalPages}}{{if .SearchQuery}}&search={{.SearchQuery}}{{end}}{{if .MinPrice}}&min_price={{.MinPrice}}{{end}}{{if .MaxPrice}}&max_price={{.MaxPrice}}{{end}}">Последняя »</a>
					{{else}}
						<span class="disabled">Следующая ›</span>
						<span class="disabled">Последняя »</span>
					{{end}}
				</div>
			</div>
		</div>
		
		<!-- Модальное окно для изображений -->
		<div id="imageModal" class="modal">
			<div class="modal-content">
				<span class="close-modal" onclick="closeImageModal()">&times;</span>
				<div class="modal-header">
					<h3 class="modal-title" id="modalTitle">ШНЫРЬ НАМУТИЛ СКРИНШОТ</h3>
					<div class="modal-info" id="modalInfo">OCR Result</div>
				</div>
				<img id="modalImage" class="modal-image" src="" alt="Full size image" onclick="closeImageModal()">
				<div id="modalStructuredData" class="modal-structured-data"></div>
			</div>
		</div>
		
		<!-- Модальное окно для детального просмотра -->
		<div id="detailModal" class="modal">
			<div class="modal-content">
				<span class="close" onclick="closeDetailModal()">&times;</span>
				<div class="modal-title" id="detailModalTitle"></div>
				
				<div class="modal-section">
					<h3>📸 Скриншот</h3>
					<img id="detailModalImage" class="modal-image" alt="Screenshot">
				</div>
				
				<div class="modal-section">
					<h3>📋 Структурированные данные</h3>
					<div id="detailModalStructuredData"></div>
				</div>
				
				<div class="modal-section">
					<h3>📄 Сырой текст</h3>
					<div id="detailModalRawText"></div>
				</div>
				
				<div class="modal-section">
					<h3>📝 Отладочная информация</h3>
					<div id="detailModalDebugInfo"></div>
				</div>
			</div>
		</div>
		
		<script>
			function openImageModal(imageData, id, info, hasItems, ...items) {
				const modal = document.getElementById('imageModal');
				const modalImage = document.getElementById('modalImage');
				const modalTitle = document.getElementById('modalTitle');
				const modalInfo = document.getElementById('modalInfo');
				const modalStructuredData = document.getElementById('modalStructuredData');
				
				modalImage.src = 'data:image/png;base64,' + imageData;
				modalTitle.textContent = 'ШНЫРЬ НАМУТИЛ СКРИНШОТ #' + id;
				modalInfo.textContent = info;
				
				// Создаем таблицу структурированных данных
				if (hasItems && items.length > 0) {
					// Находим самые дешевые предметы для каждого уровня улучшения
					const enhancementGroups = {};
					items.forEach(item => {
						if (item.enhancement && item.price) {
							const price = parseFloat(item.price.replace(/[^\d.]/g, ''));
							if (!isNaN(price)) {
								if (!enhancementGroups[item.enhancement]) {
									enhancementGroups[item.enhancement] = [];
								}
								enhancementGroups[item.enhancement].push({...item, priceValue: price});
							}
						}
					});
					
					// Находим самые дешевые предметы
					const cheapestItems = new Set();
					Object.values(enhancementGroups).forEach(group => {
						if (group.length > 0) {
							const cheapest = group.reduce((min, item) => 
								item.priceValue < min.priceValue ? item : min
							);
							cheapestItems.add(cheapest);
						}
					});
					
					let tableHTML = '<h4 style="margin: 0 0 10px 0; color: #333; font-size: 1.1em;">📋 Структурированные данные:</h4>';
					tableHTML += '<table class="modal-structured-table">';
					tableHTML += '<tr><th>Title</th><th>Title Short</th><th>Enhancement</th><th>Price</th><th>Package</th><th>Owner</th></tr>';
					
					items.forEach(item => {
						const isCheapest = cheapestItems.has(item);
						const rowClass = isCheapest ? 'cheapest-item' : '';
						tableHTML += '<tr class="' + rowClass + '">';
						tableHTML += '<td>' + (item.title || '') + '</td>';
						tableHTML += '<td>' + (item.titleShort || '') + '</td>';
						tableHTML += '<td>' + (item.enhancement || '') + '</td>';
						tableHTML += '<td>' + formatPrice(item.price || '') + '</td>';
						tableHTML += '<td>' + (item.package ? '✔️' : '') + '</td>';
						tableHTML += '<td>' + (item.owner || '') + '</td>';
						tableHTML += '</tr>';
					});
					
					tableHTML += '</table>';
					modalStructuredData.innerHTML = tableHTML;
					modalStructuredData.style.display = 'block';
				} else {
					modalStructuredData.innerHTML = '<p style="margin: 0; color: #666; font-style: italic;">Нет структурированных данных</p>';
					modalStructuredData.style.display = 'block';
				}
				
				modal.style.display = 'block';
				document.body.style.overflow = 'hidden'; // Блокируем скролл страницы
			}
			
			function closeImageModal() {
				const modal = document.getElementById('imageModal');
				modal.style.display = 'none';
				document.body.style.overflow = 'auto'; // Возвращаем скролл страницы
			}
			
			// Закрытие модального окна при клике вне изображения
			document.getElementById('imageModal').addEventListener('click', function(e) {
				if (e.target === this) {
					closeImageModal();
				}
			});
			
			// Закрытие модального окна по клавише Escape
			document.addEventListener('keydown', function(e) {
				if (e.key === 'Escape') {
					closeImageModal();
				}
			});

			function openDetailModalFromData(element) {
				const rawText = element.getAttribute('data-raw-text');
				const id = element.getAttribute('data-id');
				const imageData = element.getAttribute('data-image');
				const debugInfo = element.getAttribute('data-debug');
				const hasItems = element.getAttribute('data-items') === 'true';
				const structuredItemsJson = element.getAttribute('data-structured-items');
				
				console.log('openDetailModalFromData called with:', {rawText, id, imageData, debugInfo, hasItems, structuredItemsJson});
				
				let items = [];
				if (hasItems && structuredItemsJson) {
					try {
						items = JSON.parse(structuredItemsJson);
						console.log('Parsed items:', items);
					} catch (e) {
						console.error('Error parsing structured items:', e);
						items = [];
					}
				}
				
				openDetailModal(rawText, id, imageData, debugInfo, hasItems, ...items);
			}

			function openDetailModal(text, id, imageData, debugInfo, hasItems, ...items) {
				console.log('openDetailModal called with:', {text, id, imageData, debugInfo, hasItems, items});
				
				const modalTitle = document.getElementById('detailModalTitle');
				const modalImage = document.getElementById('detailModalImage');
				const modalDebugInfo = document.getElementById('detailModalDebugInfo');
				const modalStructuredData = document.getElementById('detailModalStructuredData');
				const modalRawText = document.getElementById('detailModalRawText');
				
				console.log('Found elements:', {
					modalTitle: !!modalTitle,
					modalImage: !!modalImage,
					modalDebugInfo: !!modalDebugInfo,
					modalStructuredData: !!modalStructuredData,
					modalRawText: !!modalRawText
				});
				
				// Устанавливаем заголовок
				modalTitle.textContent = 'ШНЫРЬ НАМУТИЛ СКРИНШОТ #' + id;
				
				// Загружаем изображение
				if (imageData && imageData !== '') {
					modalImage.src = 'data:image/png;base64,' + imageData;
					modalImage.style.display = 'block';
					console.log('Image src set to:', modalImage.src.substring(0, 50) + '...');
				} else {
					modalImage.style.display = 'none';
				}
				
				// Устанавливаем debug info
				modalDebugInfo.textContent = debugInfo || 'Нет debug информации';
				
				// Устанавливаем сырой текст
				modalRawText.textContent = text || 'Нет данных';
				
				console.log('hasItems:', hasItems, 'items length:', items.length);
				
				// Обрабатываем структурированные данные
				if (hasItems && items.length > 0) {
					console.log('Processing items:', items);
					const cheapestItems = new Set();
					
					// Группируем предметы по уровню улучшения и package
					const enhancementGroups = {};
					items.forEach(item => {
						const enhancement = item.enhancement || '';
						const isPackage = item.package || false;
						const groupKey = enhancement + '_' + (isPackage ? 'package' : 'nopackage');
						
						if (!enhancementGroups[groupKey]) {
							enhancementGroups[groupKey] = [];
						}
						enhancementGroups[groupKey].push(item);
					});
					
					// Находим самые дешевые предметы в каждой группе
					Object.values(enhancementGroups).forEach(group => {
						if (group.length > 0) {
							const cheapest = group.reduce((min, item) => {
								const priceValue = parseFloat((item.price || '0').replace(/[^\d.]/g, ''));
								const minPriceValue = parseFloat((min.price || '0').replace(/[^\d.]/g, ''));
								return priceValue < minPriceValue ? item : min;
							});
							cheapestItems.add(cheapest);
						}
					});
					
					let tableHTML = '<table class="structured-table">';
					tableHTML += '<thead><tr><th>Название</th><th>Краткое название</th><th>Улучшение</th><th>Цена</th><th>Пакет</th><th>Владелец</th></tr></thead>';
					tableHTML += '<tbody>';
					
					items.forEach(item => {
						const isCheapest = cheapestItems.has(item);
						const rowClass = isCheapest ? (item.package ? 'cheapest-package' : 'cheapest') : '';
						tableHTML += '<tr class="' + rowClass + '">';
						tableHTML += '<td>' + (item.title || '') + '</td>';
						tableHTML += '<td>' + (item.titleShort || '') + '</td>';
						tableHTML += '<td>' + (item.enhancement || '') + '</td>';
						tableHTML += '<td>' + formatPrice(item.price || '') + '</td>';
						tableHTML += '<td>' + (item.package ? '✔️' : '❌') + '</td>';
						tableHTML += '<td>' + (item.owner || '') + '</td>';
						tableHTML += '</tr>';
					});
					
					tableHTML += '</tbody></table>';
					console.log('Generated table HTML:', tableHTML);
					modalStructuredData.innerHTML = tableHTML;
					console.log('Table HTML set to modalStructuredData');
				} else {
					console.log('No items to display');
					modalStructuredData.innerHTML = '<p>Нет структурированных данных</p>';
				}
				
				// Показываем модальное окно
				const detailModal = document.getElementById('detailModal');
				detailModal.style.display = 'block';
				document.body.style.overflow = 'hidden';
				console.log('Modal displayed');
			}
			
			function closeDetailModal() {
				const detailModal = document.getElementById('detailModal');
				detailModal.style.display = 'none';
				document.body.style.overflow = 'auto';
			}
			
			// Закрытие модального окна при клике вне информации
			document.getElementById('detailModal').addEventListener('click', function(e) {
				if (e.target === this) {
					closeDetailModal();
				}
			});
			
			// Закрытие модального окна по клавише Escape
			document.addEventListener('keydown', function(e) {
				if (e.key === 'Escape') {
					closeDetailModal();
				}
			});
			
			// Функция для форматирования цены с пробелами
			function formatPrice(price) {
				if (!price) return '';
				// Убираем все нецифровые символы
				const cleanPrice = price.replace(/[^\d.]/g, '');
				if (!cleanPrice) return price;
				
				// Добавляем пробелы каждые 3 цифры справа
				let result = '';
				for (let i = 0; i < cleanPrice.length; i++) {
					if (i > 0 && (cleanPrice.length - i) % 3 === 0) {
						result += ' ';
					}
					result += cleanPrice[i];
				}
				return result;
			}

			// Функция для выделения самых дешевых предметов в главной таблице
			function highlightCheapestItems() {
				console.log('highlightCheapestItems called');
				const structuredTables = document.querySelectorAll('.structured-table table');
				console.log('Found structured tables:', structuredTables.length);
				
				structuredTables.forEach(function(table, tableIndex) {
					console.log('Processing table ' + tableIndex);
					const rows = table.querySelectorAll('tr:not(:first-child)'); // Исключаем заголовок
					console.log('Found ' + rows.length + ' data rows');
					const enhancementGroups = {};
					
					// Группируем предметы по уровню улучшения и package
					rows.forEach(function(row, rowIndex) {
						const cells = row.querySelectorAll('td');
						if (cells.length >= 5) {
							const enhancement = cells[2].textContent.trim();
							const price = cells[3].textContent.trim();
							const package = cells[4].textContent.trim();
							const priceValue = parseFloat(price.replace(/[^\d]/g, ''));
							
							console.log('Row ' + rowIndex + ': enhancement="' + enhancement + '", price="' + price + '", package="' + package + '", priceValue=' + priceValue);
							
							if (enhancement && !isNaN(priceValue)) {
								// Создаем ключ группы: enhancement + package
								const groupKey = enhancement + '_' + (package.includes('✔️') ? 'package' : 'nopackage');
								if (!enhancementGroups[groupKey]) {
									enhancementGroups[groupKey] = [];
								}
								enhancementGroups[groupKey].push({
									row: row, 
									priceValue: priceValue, 
									isPackage: package.includes('✔️')
								});
							}
						}
					});
					
					console.log('Enhancement groups:', enhancementGroups);
					
					// Находим и выделяем самые дешевые предметы в каждой группе
					Object.keys(enhancementGroups).forEach(function(groupKey) {
						const group = enhancementGroups[groupKey];
						console.log('Processing group ' + groupKey + ' with ' + group.length + ' items');
						if (group.length > 0) {
							const cheapest = group.reduce(function(min, item) {
								return item.priceValue < min.priceValue ? item : min;
							});
							console.log('Cheapest in group ' + groupKey + ': price=' + cheapest.priceValue + ', isPackage=' + cheapest.isPackage);
							
							// Применяем соответствующий класс в зависимости от package
							if (cheapest.isPackage) {
								cheapest.row.classList.add('cheapest-package');
							} else {
								cheapest.row.classList.add('cheapest');
							}
						}
					});
				});
			}
			
			// Вызываем функцию после загрузки страницы
			document.addEventListener('DOMContentLoaded', highlightCheapestItems);
		</script>
		</body></html>`

		t, err := template.New("web").Funcs(template.FuncMap{
			"base64encode": func(data []byte) string {
				return base64.StdEncoding.EncodeToString(data)
			},
			"jsEscape": func(s string) string {
				return strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`)
			},
			"formatPrice": func(price string) string {
				// Убираем все нецифровые символы
				cleanPrice := strings.ReplaceAll(strings.ReplaceAll(price, ",", ""), " ", "")
				if cleanPrice == "" {
					return price
				}

				// Добавляем пробелы каждые 3 цифры справа
				var result string
				for i, char := range cleanPrice {
					if i > 0 && (len(cleanPrice)-i)%3 == 0 {
						result += " "
					}
					result += string(char)
				}
				return result
			},
			"sequence": func(current, total int) []int {
				var pages []int
				start := current - 2
				if start < 1 {
					start = 1
				}
				end := current + 2
				if end > total {
					end = total
				}
				for i := start; i <= end; i++ {
					pages = append(pages, i)
				}
				return pages
			},
		}).Parse(tmpl)
		if err != nil {
			http.Error(w, "Template error", 500)
			return
		}
		t.Execute(w, pageData)
	})

	fmt.Printf("🚀 ШНЫРЬ v0.1 запущен на порту %s\n", port)
	fmt.Printf("📊 База данных: %s\n", dbDSN)
	fmt.Printf("🌐 Откройте http://localhost:%s в браузере\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
