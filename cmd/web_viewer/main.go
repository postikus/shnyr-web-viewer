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
	"time"

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
	Count       string
	Category    string
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
	Results                 []OCRResult
	CurrentPage             int
	TotalPages              int
	TotalCount              int
	HasPrev                 bool
	HasNext                 bool
	PrevPage                int
	NextPage                int
	SearchQuery             string
	MinPrice                string
	MaxPrice                string
	ActiveTab               string
	ItemSearch              string
	ItemResults             []StructuredItem
	CategoryBuyConsumables  bool
	CategoryBuyEquipment    bool
	CategorySellConsumables bool
	CategorySellEquipment   bool
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

	// Получаем хост из переменной окружения
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
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
	log.Printf("Запускаем сервер на %s:%s", host, port)

	// Настройка статических файлов
	staticPath := "static"
	if _, err := os.Stat("static"); os.IsNotExist(err) {
		// Если нет, пробуем относительный путь
		staticPath = "cmd/web_viewer/static"
	}

	fs := http.FileServer(http.Dir(staticPath))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Получаем параметры пагинации и поиска
		pageStr := r.URL.Query().Get("page")
		searchQuery := r.URL.Query().Get("search")
		minPrice := r.URL.Query().Get("min_price")
		maxPrice := r.URL.Query().Get("max_price")
		activeTab := r.URL.Query().Get("tab")
		itemSearch := r.URL.Query().Get("item_search")

		if activeTab == "" {
			activeTab = "main"
		}

		page := 1
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		// Получаем фильтры категорий для поиска по предмету
		categoryBuyConsumables := r.URL.Query().Get("category_buy_consumables") == "1"
		categoryBuyEquipment := r.URL.Query().Get("category_buy_equipment") == "1"
		categorySellConsumables := r.URL.Query().Get("category_sell_consumables") == "1"
		categorySellEquipment := r.URL.Query().Get("category_sell_equipment") == "1"

		// Если ни одна категория не выбрана, выбираем все по умолчанию
		if !categoryBuyConsumables && !categoryBuyEquipment && !categorySellConsumables && !categorySellEquipment {
			categoryBuyConsumables = true
			categoryBuyEquipment = true
			categorySellConsumables = true
			categorySellEquipment = true
		}

		// Если есть поиск по предмету, выполняем его
		var itemResults []StructuredItem
		if itemSearch != "" {
			// Формируем список категорий для поиска
			var categories []string
			if categoryBuyConsumables {
				categories = append(categories, "'buy_consumables'")
			}
			if categoryBuyEquipment {
				categories = append(categories, "'buy_equipment'")
			}
			if categorySellConsumables {
				categories = append(categories, "'sell_consumables'")
			}
			if categorySellEquipment {
				categories = append(categories, "'sell_equipment'")
			}

			// Поиск по structured_items
			itemQuery := fmt.Sprintf(`SELECT id, ocr_result_id, title, title_short, enhancement, price, package, owner, count, category, created_at 
				FROM structured_items 
				WHERE category IN (%s) AND title LIKE ? 
				ORDER BY CAST(REPLACE(REPLACE(price, ',', ''), ' ', '') AS DECIMAL(10,2)), created_at DESC`, strings.Join(categories, ", "))

			itemRows, err := db.Query(itemQuery, "%"+itemSearch+"%")
			if err != nil {
				http.Error(w, "DB error", 500)
				return
			}
			defer itemRows.Close()

			for itemRows.Next() {
				var item StructuredItem
				if err := itemRows.Scan(&item.ID, &item.OCRResultID, &item.Title, &item.TitleShort, &item.Enhancement, &item.Price, &item.Package, &item.Owner, &item.Count, &item.Category, &item.CreatedAt); err == nil {
					itemResults = append(itemResults, item)
				}
			}
		}

		// Если активна вкладка поиска по предмету и есть результаты, показываем только их
		if activeTab == "item_search" && itemSearch != "" {
			// Подготавливаем данные для шаблона
			pageData := PageData{
				ActiveTab:               activeTab,
				ItemSearch:              itemSearch,
				ItemResults:             itemResults,
				CategoryBuyConsumables:  categoryBuyConsumables,
				CategoryBuyEquipment:    categoryBuyEquipment,
				CategorySellConsumables: categorySellConsumables,
				CategorySellEquipment:   categorySellEquipment,
			}

			renderTemplate(w, pageData)
			return
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
			itemRows, err := db.Query(`SELECT id, ocr_result_id, title, title_short, enhancement, price, package, owner, count, category, created_at FROM structured_items WHERE ocr_result_id = ? ORDER BY created_at`, res.ID)
			if err == nil {
				defer itemRows.Close()
				for itemRows.Next() {
					var item StructuredItem
					if err := itemRows.Scan(&item.ID, &item.OCRResultID, &item.Title, &item.TitleShort, &item.Enhancement, &item.Price, &item.Package, &item.Owner, &item.Count, &item.Category, &item.CreatedAt); err == nil {
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
			ActiveTab:   activeTab,
		}

		renderTemplate(w, pageData)
	})

	fmt.Printf("🚀 ШНЫРЬ v0.1 запущен на порту %s\n", port)
	fmt.Printf("📊 База данных: %s\n", dbDSN)
	fmt.Printf("🌐 Откройте http://localhost:%s в браузере\n", port)

	if err := http.ListenAndServe(host+":"+port, nil); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func renderTemplate(w http.ResponseWriter, data PageData) {
	// Определяем путь к шаблонам
	templatePath := "templates/*.html"

	// Проверяем, существует ли директория templates
	if _, err := os.Stat("templates"); os.IsNotExist(err) {
		// Если нет, пробуем относительный путь
		templatePath = "cmd/web_viewer/templates/*.html"
	}

	// Загружаем все шаблоны
	tmpl, err := template.New("layout").Funcs(template.FuncMap{
		"base64encode": func(data []byte) string {
			return base64.StdEncoding.EncodeToString(data)
		},
		"jsEscape": func(s string) string {
			return strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`)
		},
		"formatDateTime": func(dateTimeStr string) string {
			// Парсим время из строки
			t, err := time.Parse("2006-01-02T15:04:05Z", dateTimeStr)
			if err != nil {
				// Если не удалось распарсить, возвращаем исходную строку
				return dateTimeStr
			}

			// Добавляем 8 часов (UTC+8)
			localTime := t.Add(8 * time.Hour)

			// Форматируем в читаемый вид
			return localTime.Format("02.01.2006 15:04:05")
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
		"formatCategory": func(category string) string {
			switch category {
			case "buy_consumables":
				return "💰 Покупай! (расходники)"
			case "buy_equipment":
				return "💰 Покупай! (экипировка)"
			case "sell_consumables":
				return "💸 Продавай! (расходники)"
			case "sell_equipment":
				return "💸 Продавай! (экипировка)"
			case "unknown":
				return "❓ Неизвестно"
			default:
				return category
			}
		},
	}).ParseGlob(templatePath)

	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Template execution error: "+err.Error(), 500)
		return
	}
}
