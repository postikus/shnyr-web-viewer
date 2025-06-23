package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

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

		if searchQuery != "" {
			// Поиск по структурированным данным
			countQuery = `SELECT COUNT(DISTINCT ocr.id) FROM ocr_results ocr 
				LEFT JOIN structured_items si ON ocr.id = si.ocr_result_id 
				WHERE si.title LIKE ? OR si.owner LIKE ? OR si.price LIKE ? OR si.title_short LIKE ?`
			dataQuery = `SELECT DISTINCT ocr.id, ocr.image_path, ocr.image_data, ocr.ocr_text, ocr.debug_info, ocr.json_data, ocr.raw_text, ocr.created_at 
				FROM ocr_results ocr 
				LEFT JOIN structured_items si ON ocr.id = si.ocr_result_id 
				WHERE si.title LIKE ? OR si.owner LIKE ? OR si.price LIKE ? OR si.title_short LIKE ? 
				ORDER BY ocr.created_at DESC LIMIT ? OFFSET ?`

			searchPattern := "%" + searchQuery + "%"
			args = []interface{}{searchPattern, searchPattern, searchPattern, searchPattern}
		} else {
			// Без поиска
			countQuery = "SELECT COUNT(*) FROM ocr_results"
			dataQuery = `SELECT id, image_path, image_data, ocr_text, debug_info, json_data, raw_text, created_at FROM ocr_results ORDER BY created_at DESC LIMIT ? OFFSET ?`
		}

		// Получаем общее количество записей
		var totalCount int
		var countArgs []interface{}
		if searchQuery != "" {
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
		if searchQuery != "" {
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
					background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
					min-height: 100vh;
					color: #333;
				}
				
				.container {
					width: 100%;
					margin: 0;
					background: white;
					border-radius: 0;
					box-shadow: none;
					overflow: hidden;
				}
				
				.header {
					background: linear-gradient(135deg, #4CAF50, #45a049);
					color: white;
					padding: 15px;
					text-align: center;
					margin: 0;
					position: fixed;
					top: 0;
					left: 0;
					right: 0;
					z-index: 1000;
					box-shadow: 0 2px 10px rgba(0,0,0,0.3);
				}
				
				.header h2 {
					margin: 0 0 10px 0;
					font-size: 1.5em;
					font-weight: 300;
					text-shadow: 0 2px 4px rgba(0,0,0,0.3);
				}
				
				.search-container {
					display: flex;
					flex-direction: column;
					gap: 8px;
					max-width: 100%;
					margin: 0 auto;
				}
				
				.search-input {
					width: 100%;
					padding: 12px 16px;
					border: none;
					border-radius: 25px;
					font-size: 16px;
					outline: none;
					box-shadow: 0 2px 8px rgba(0,0,0,0.1);
					transition: box-shadow 0.3s ease;
				}
				
				.search-input:focus {
					box-shadow: 0 4px 12px rgba(0,0,0,0.2);
				}
				
				.search-button {
					padding: 12px 20px;
					background: rgba(255,255,255,0.2);
					border: 2px solid rgba(255,255,255,0.3);
					color: white;
					border-radius: 25px;
					cursor: pointer;
					font-size: 16px;
					transition: all 0.3s ease;
					backdrop-filter: blur(10px);
					width: 100%;
				}
				
				.search-button:hover {
					background: rgba(255,255,255,0.3);
					border-color: rgba(255,255,255,0.5);
					transform: translateY(-2px);
				}
				
				.clear-button {
					padding: 10px 16px;
					background: rgba(255,255,255,0.1);
					border: 2px solid rgba(255,255,255,0.2);
					color: white;
					border-radius: 25px;
					cursor: pointer;
					font-size: 14px;
					transition: all 0.3s ease;
					text-decoration: none;
					text-align: center;
					width: 100%;
				}
				
				.clear-button:hover {
					background: rgba(255,255,255,0.2);
					border-color: rgba(255,255,255,0.4);
				}
				
				.content {
					padding: 15px;
					margin-top: 140px;
				}
				
				.stats { 
					text-align: center; 
					margin: 15px 0; 
					padding: 12px;
					background: #f8f9fa;
					border-radius: 8px;
					border-left: 4px solid #4CAF50;
					font-size: 0.9em;
					color: #555;
				}
				
				.search-info {
					text-align: center;
					margin: 10px 0;
					padding: 10px;
					background: #e8f5e8;
					border-radius: 8px;
					color: #2e7d32;
					font-weight: 500;
					font-size: 0.9em;
				}
				
				/* Мобильная таблица */
				.mobile-table {
					display: block;
					width: 100%;
					overflow-x: auto;
					-webkit-overflow-scrolling: touch;
				}
				
				table { 
					border-collapse: collapse; 
					width: 100%; 
					margin-bottom: 20px;
					background: white;
					border-radius: 8px;
					overflow: hidden;
					box-shadow: 0 3px 10px rgba(0,0,0,0.08);
					min-width: 800px; /* Минимальная ширина для горизонтального скролла */
				}
				
				th, td { 
					border: 1px solid #e0e0e0; 
					padding: 8px; 
					text-align: left; 
					vertical-align: top;
					font-size: 0.8em;
				}
				
				th { 
					background: linear-gradient(135deg, #4CAF50, #45a049);
					color: white;
					font-weight: 600;
					text-transform: uppercase;
					font-size: 0.75em;
					letter-spacing: 0.3px;
					white-space: nowrap;
				}
				
				tr:nth-child(even) {
					background-color: #f9f9f9;
				}
				
				tr:hover {
					background-color: #f0f8ff;
					transition: background-color 0.3s ease;
				}
				
				.image-cell img {
					max-width: 150px;
					max-height: 100px;
					border-radius: 6px;
					box-shadow: 0 2px 4px rgba(0,0,0,0.1);
					border: 1px solid #e0e0e0;
				}
				
				.text-cell pre {
					max-width: 200px;
					max-height: 150px;
					overflow: auto;
					background: #f8f9fa;
					padding: 8px;
					border-radius: 4px;
					border: 1px solid #e0e0e0;
					font-size: 0.7em;
					line-height: 1.3;
					margin: 0;
				}
				
				.structured-table {
					border: 1px solid #ddd;
					font-size: 9px;
					border-radius: 4px;
					overflow: visible;
					box-shadow: 0 1px 3px rgba(0,0,0,0.1);
					max-width: 300px;
				}
				
				.structured-table table {
					width: 100%;
					margin: 0;
					box-shadow: none;
					min-width: auto;
				}
				
				.structured-table th {
					background: #6c757d;
					color: white;
					padding: 4px 6px;
					font-size: 8px;
				}
				
				.structured-table td {
					padding: 3px 4px;
					border: 1px solid #e0e0e0;
					font-size: 8px;
					white-space: normal;
					word-wrap: break-word;
				}
				
				.structured-table tr:nth-child(even) {
					background-color: #f8f9fa;
				}
				
				.pagination { 
					text-align: center; 
					margin: 20px 0; 
					padding: 15px;
					background: #f8f9fa;
					border-radius: 8px;
				}
				
				.pagination a, .pagination span { 
					display: inline-block; 
					padding: 8px 12px; 
					text-decoration: none; 
					border: 2px solid #4CAF50; 
					margin: 2px; 
					border-radius: 6px;
					font-weight: 500;
					transition: all 0.3s ease;
					min-width: 35px;
					font-size: 0.9em;
				}
				
				.pagination a { 
					background: white;
					color: #4CAF50;
				}
				
				.pagination a:hover { 
					background: #4CAF50; 
					color: white;
					transform: translateY(-1px);
					box-shadow: 0 2px 6px rgba(76, 175, 80, 0.3);
				}
				
				.pagination .current { 
					background: #4CAF50; 
					color: white;
					font-weight: bold;
				}
				
				.pagination .disabled { 
					color: #ccc; 
					border-color: #ccc;
					pointer-events: none;
					background: #f5f5f5;
				}
				
				.no-data {
					text-align: center;
					padding: 15px;
					color: #666;
					font-style: italic;
					background: #f8f9fa;
					border-radius: 4px;
					border: 1px dashed #ccc;
					font-size: 0.8em;
				}
				
				.id-cell {
					font-weight: bold;
					color: #4CAF50;
					text-align: center;
					font-size: 0.8em;
				}
				
				.date-cell {
					font-size: 0.75em;
					color: #666;
					white-space: nowrap;
				}
				
				/* Планшеты */
				@media (min-width: 768px) {
					.header {
						padding: 20px;
					}
					
					.header h2 {
						font-size: 2em;
						margin-bottom: 15px;
					}
					
					.search-container {
						flex-direction: row;
						gap: 10px;
						max-width: 500px;
					}
					
					.search-input {
						flex: 1;
					}
					
					.search-button, .clear-button {
						width: auto;
					}
					
					.content {
						padding: 20px;
						margin-top: 120px;
					}
					
					.stats {
						font-size: 1.1em;
						padding: 15px;
					}
					
					th, td {
						padding: 12px;
						font-size: 0.9em;
					}
					
					.image-cell img {
						max-width: 200px;
						max-height: 150px;
					}
					
					.text-cell pre {
						max-width: 250px;
						font-size: 0.8em;
					}
					
					.structured-table {
						max-width: 400px;
						font-size: 10px;
					}
					
					.structured-table th {
						font-size: 9px;
						padding: 6px 8px;
					}
					
					.structured-table td {
						font-size: 9px;
						padding: 4px 6px;
					}
				}
				
				/* Десктопы */
				@media (min-width: 1024px) {
					.header h2 {
						font-size: 2.5em;
					}
					
					.content {
						padding: 30px;
					}
					
					th, td {
						padding: 15px;
						font-size: 1em;
					}
					
					.image-cell img {
						max-width: 300px;
						max-height: 200px;
					}
					
					.text-cell pre {
						max-width: 300px;
						font-size: 0.85em;
					}
					
					.structured-table {
						max-width: 500px;
						font-size: 11px;
					}
					
					.structured-table th {
						font-size: 10px;
						padding: 8px;
					}
					
					.structured-table td {
						font-size: 10px;
						padding: 6px 8px;
					}
					
					table {
						min-width: auto;
					}
				}
				
				/* Большие экраны */
				@media (min-width: 1200px) {
					.container {
						max-width: 1400px;
						margin: 0 auto;
					}
					
					.text-cell pre {
						max-width: 350px;
					}
					
					.image-cell img {
						max-width: 350px;
					}
				}
			</style>
		</head>
		<body>
		<div class="container">
			<div class="header">
				<h2>👓 ШНЫРЬ v0.1</h2>
				<form method="GET" action="/" class="search-container">
					<input type="text" name="search" value="{{.SearchQuery}}" placeholder="Поиск по названию, владельцу, цене..." class="search-input">
					<button type="submit" class="search-button">🔍 Поиск</button>
					{{if .SearchQuery}}
					<a href="/" class="clear-button">❌ Очистить</a>
					{{end}}
				</form>
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
					<th>ID</th>
					<th>Изображение</th>
					<th>Structured Data</th>
					<th>Raw Text</th>
					<th>Debug Info</th>
					<th>JSON Data</th>
					<th>Created</th>
				</tr>
				{{range .Results}}
				<tr>
				<td class="id-cell">{{.ID}}</td>
				<td class="image-cell">
					{{if .ImageData}}
					<img src="data:image/png;base64,{{base64encode .ImageData}}" />
					{{else}}
					<div class="no-data">No image data</div>
					{{end}}
				</td>
				<td>
					{{if .Items}}
					<div class="structured-table">
					<table>
					<tr><th>Title</th><th>Title Short</th><th>Enhancement</th><th>Price</th><th>Owner</th></tr>
					{{range .Items}}
					<tr>
					<td>{{.Title}}</td>
					<td>{{.TitleShort}}</td>
					<td>{{.Enhancement}}</td>
					<td>{{.Price}}</td>
					<td>{{.Owner}}</td>
					</tr>
					{{end}}
					</table>
					</div>
					{{else}}
					<div class="no-data">No structured data</div>
					{{end}}
				</td>
				<td class="text-cell">
					{{if .RawText}}
					<pre>{{.RawText}}</pre>
					{{else}}
					<div class="no-data">No raw text</div>
					{{end}}
				</td>
				<td class="text-cell">
					{{if .DebugInfo}}
					<pre>{{.DebugInfo}}</pre>
					{{else}}
					<div class="no-data">No debug info</div>
					{{end}}
				</td>
				<td class="text-cell">
					{{if .JSONData}}
					<pre>{{.JSONData}}</pre>
					{{else}}
					<div class="no-data">No JSON data</div>
					{{end}}
				</td>
				<td class="date-cell">{{.CreatedAt}}</td>
				</tr>
				{{end}}
				</table>
				</div>

				<div class="pagination">
					{{if .HasPrev}}
						<a href="?page=1{{if .SearchQuery}}&search={{.SearchQuery}}{{end}}">« Первая</a>
						<a href="?page={{.PrevPage}}{{if .SearchQuery}}&search={{.SearchQuery}}{{end}}">‹ Предыдущая</a>
					{{else}}
						<span class="disabled">« Первая</span>
						<span class="disabled">‹ Предыдущая</span>
					{{end}}
					
					{{range $i := sequence .CurrentPage .TotalPages}}
						{{if eq $i $.CurrentPage}}
							<span class="current">{{$i}}</span>
						{{else}}
							<a href="?page={{$i}}{{if $.SearchQuery}}&search={{$.SearchQuery}}{{end}}">{{$i}}</a>
						{{end}}
					{{end}}
					
					{{if .HasNext}}
						<a href="?page={{.NextPage}}{{if .SearchQuery}}&search={{.SearchQuery}}{{end}}">Следующая ›</a>
						<a href="?page={{.TotalPages}}{{if .SearchQuery}}&search={{.SearchQuery}}{{end}}">Последняя »</a>
					{{else}}
						<span class="disabled">Следующая ›</span>
						<span class="disabled">Последняя »</span>
					{{end}}
				</div>
			</div>
		</div>
		</body></html>`

		t, err := template.New("web").Funcs(template.FuncMap{
			"base64encode": func(data []byte) string {
				return base64.StdEncoding.EncodeToString(data)
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
