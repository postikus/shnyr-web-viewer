package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

// Prometheus метрики для отслеживания цен gold coin
var (
	// Тестовая метрика для проверки работы
	testMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "shnyr_test_metric",
			Help: "Тестовая метрика для проверки работы Prometheus",
		},
	)

	goldCoinAvgPrice = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gold_coin_avg_min_3_prices",
			Help: "Среднее из 3 минимальных цен для gold coin",
		},
		[]string{"category"},
	)

	goldCoinMinPrice = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gold_coin_min_price",
			Help: "Минимальная цена для gold coin",
		},
		[]string{"category"},
	)

	goldCoinMaxPrice = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gold_coin_max_price_of_min_3",
			Help: "Максимальная из 3 минимальных цен для gold coin",
		},
		[]string{"category"},
	)

	goldCoinPriceCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gold_coin_prices_count",
			Help: "Количество цен для gold coin",
		},
		[]string{"category"},
	)
)

func init() {
	// Регистрируем метрики
	prometheus.MustRegister(testMetric)
	prometheus.MustRegister(goldCoinAvgPrice)
	prometheus.MustRegister(goldCoinMinPrice)
	prometheus.MustRegister(goldCoinMaxPrice)
	prometheus.MustRegister(goldCoinPriceCount)

	// Устанавливаем тестовую метрику
	testMetric.Set(1.0)
}

// updateGoldCoinMetrics обновляет метрики для gold coin
func updateGoldCoinMetrics(db *sql.DB) {
	log.Printf("🔄 Обновление метрик gold coin...")

	query := `
	WITH gold_coin_ocr AS (
		SELECT DISTINCT ocr.id as ocr_id
		FROM octopus.ocr_results ocr
		INNER JOIN octopus.structured_items si ON ocr.id = si.ocr_result_id
		WHERE si.title = 'gold coin' 
		  AND si.category = 'buy_consumables'
	),
	price_analysis AS (
		SELECT 
			gco.ocr_id,
			si.id as structured_item_id,
			si.title,
			si.category,
			si.price,
			CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(15,2)) as price_numeric
		FROM gold_coin_ocr gco
		INNER JOIN octopus.structured_items si ON gco.ocr_id = si.ocr_result_id
		WHERE si.price IS NOT NULL 
		  AND si.price != ''
		  AND CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(15,2)) > 0
	),
	top_3_prices AS (
		SELECT 
			ocr_id,
			title,
			category,
			price,
			price_numeric,
			ROW_NUMBER() OVER (PARTITION BY ocr_id ORDER BY price_numeric ASC) as price_rank
		FROM price_analysis
	),
	avg_min_3_prices AS (
		SELECT 
			ocr_id,
			title,
			category,
			COUNT(*) as prices_count,
			AVG(price_numeric) as avg_min_3_prices,
			MIN(price_numeric) as min_price,
			MAX(price_numeric) as max_price_of_min_3
		FROM top_3_prices
		WHERE price_rank <= 3
		GROUP BY ocr_id, title, category
	)
	SELECT 
		category,
		COUNT(*) as total_records,
		AVG(avg_min_3_prices) as avg_price,
		MIN(min_price) as min_price,
		MAX(max_price_of_min_3) as max_price,
		SUM(prices_count) as total_prices
	FROM avg_min_3_prices
	GROUP BY category
	ORDER BY category
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("❌ Ошибка получения метрик gold coin: %v", err)
		return
	}
	defer rows.Close()

	metricsCount := 0
	for rows.Next() {
		var category string
		var totalRecords int
		var avgPrice, minPrice, maxPrice float64
		var totalPrices int

		err := rows.Scan(&category, &totalRecords, &avgPrice, &minPrice, &maxPrice, &totalPrices)
		if err != nil {
			log.Printf("❌ Ошибка сканирования метрик: %v", err)
			continue
		}

		// Обновляем метрики
		goldCoinAvgPrice.WithLabelValues(category).Set(avgPrice)
		goldCoinMinPrice.WithLabelValues(category).Set(minPrice)
		goldCoinMaxPrice.WithLabelValues(category).Set(maxPrice)
		goldCoinPriceCount.WithLabelValues(category).Set(float64(totalPrices))

		log.Printf("📊 Обновлена метрика для категории %s: avg=%.2f, min=%.2f, max=%.2f, count=%d",
			category, avgPrice, minPrice, maxPrice, totalPrices)
		metricsCount++
	}

	log.Printf("✅ Обновлено %d метрик gold coin", metricsCount)
}

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

type ItemsListItem struct {
	ID            int
	Name          string
	Category      string
	MinPrice      sql.NullFloat64
	MinPriceValue float64
	MinPriceValid bool
	CreatedAt     string
}

type Status struct {
	ID            int
	CurrentStatus string
	UpdatedAt     string
}

type Action struct {
	ID        int
	Action    string
	Executed  bool
	CreatedAt string
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
	ItemsList               []ItemsListItem
	CategoryBuyConsumables  bool
	CategoryBuyEquipment    bool
	CategorySellConsumables bool
	CategorySellEquipment   bool
	Status                  Status
	RecentActions           []Action
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

func getItemsList(db *sql.DB) ([]ItemsListItem, error) {
	rows, err := db.Query("SELECT id, name, category, min_price, created_at FROM items_list ORDER BY category, id")
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса items_list: %v", err)
	}
	defer rows.Close()

	var items []ItemsListItem
	for rows.Next() {
		var item ItemsListItem
		err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.MinPrice, &item.CreatedAt)
		if err != nil {
			log.Printf("Ошибка сканирования items_list: %v, пропускаем запись", err)
			continue // Пропускаем проблемную запись
		}
		item.MinPriceValue = item.MinPrice.Float64
		item.MinPriceValid = item.MinPrice.Valid
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по items_list: %v", err)
	}

	return items, nil
}

func getCurrentStatus(db *sql.DB) (Status, error) {
	var status Status
	err := db.QueryRow("SELECT id, current_status, updated_at FROM status ORDER BY id DESC LIMIT 1").Scan(&status.ID, &status.CurrentStatus, &status.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Если нет записей, возвращаем статус по умолчанию
			return Status{ID: 0, CurrentStatus: "unknown", UpdatedAt: ""}, nil
		}
		return Status{}, fmt.Errorf("ошибка получения статуса: %v", err)
	}
	return status, nil
}

func getRecentActions(db *sql.DB, limit int) ([]Action, error) {
	rows, err := db.Query("SELECT id, action, executed, created_at FROM actions ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса действий: %v", err)
	}
	defer rows.Close()

	var actions []Action
	for rows.Next() {
		var action Action
		err := rows.Scan(&action.ID, &action.Action, &action.Executed, &action.CreatedAt)
		if err != nil {
			log.Printf("Ошибка сканирования actions: %v, пропускаем запись", err)
			continue
		}
		actions = append(actions, action)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по actions: %v", err)
	}

	return actions, nil
}

func addAction(db *sql.DB, action string) error {
	_, err := db.Exec("INSERT INTO actions (action) VALUES (?)", action)
	return err
}

func addActionWithExecuted(db *sql.DB, action string, executed bool) error {
	_, err := db.Exec("INSERT INTO actions (action, executed) VALUES (?, ?)", action, executed)
	return err
}

func updateActionExecuted(db *sql.DB, actionID int, executed bool) error {
	_, err := db.Exec("UPDATE actions SET executed = ? WHERE id = ?", executed, actionID)
	return err
}

func getLatestPendingAction(db *sql.DB) (*Action, error) {
	var action Action
	err := db.QueryRow("SELECT id, action, executed, created_at FROM actions WHERE executed = FALSE ORDER BY created_at DESC LIMIT 1").Scan(&action.ID, &action.Action, &action.Executed, &action.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &action, nil
}

func updateStatus(db *sql.DB, status string) error {
	_, err := db.Exec("INSERT INTO status (current_status) VALUES (?)", status)
	return err
}

func updateLatestPendingAction(db *sql.DB) error {
	action, err := getLatestPendingAction(db)
	if err != nil {
		return err
	}
	if action != nil {
		err = updateActionExecuted(db, action.ID, true)
		if err != nil {
			return err
		}
	}
	return nil
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

	log.Printf("🚀 Запуск ШНЫРЬ v0.1")
	log.Printf("📋 Переменные окружения:")
	log.Printf("   PORT: %s", port)
	log.Printf("   HOST: %s", host)
	log.Printf("   DB_HOST: %s", os.Getenv("DB_HOST"))
	log.Printf("   DB_PORT: %s", os.Getenv("DB_PORT"))
	log.Printf("   DB_USER: %s", os.Getenv("DB_USER"))
	log.Printf("   DB_NAME: %s", os.Getenv("DB_NAME"))

	// Подключаемся к базе данных
	dbDSN := getDatabaseDSN()
	log.Printf("🔗 Подключение к базе данных: %s", dbDSN)

	db, err := sql.Open("mysql", dbDSN)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Проверяем подключение
	log.Printf("🔍 Проверка подключения к базе данных...")
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Ошибка проверки подключения к базе данных: %v", err)
	}

	log.Printf("✅ Успешно подключились к базе данных")
	log.Printf("🌐 Запускаем сервер на %s:%s", host, port)

	// Настройка статических файлов
	staticPath := "static"
	if _, err := os.Stat("static"); os.IsNotExist(err) {
		// Если нет, пробуем относительный путь
		staticPath = "cmd/web_viewer/static"
	}

	fs := http.FileServer(http.Dir(staticPath))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Логируем все запросы для диагностики
		log.Printf("🌐 Запрос к %s от %s", r.URL.Path, r.RemoteAddr)
		log.Printf("🌐 User-Agent: %s", r.UserAgent())

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
			// Получаем статус и действия
			status, err := getCurrentStatus(db)
			if err != nil {
				log.Printf("Ошибка получения статуса: %v", err)
				status = Status{ID: 0, CurrentStatus: "unknown", UpdatedAt: ""}
			}

			recentActions, err := getRecentActions(db, 5)
			if err != nil {
				log.Printf("Ошибка получения действий: %v", err)
				recentActions = []Action{}
			}

			// Подготавливаем данные для шаблона
			pageData := PageData{
				ActiveTab:               activeTab,
				ItemSearch:              itemSearch,
				ItemResults:             itemResults,
				CategoryBuyConsumables:  categoryBuyConsumables,
				CategoryBuyEquipment:    categoryBuyEquipment,
				CategorySellConsumables: categorySellConsumables,
				CategorySellEquipment:   categorySellEquipment,
				Status:                  status,
				RecentActions:           recentActions,
			}

			renderTemplate(w, pageData)
			return
		}

		// Получаем список предметов из items_list
		itemsList, err := getItemsList(db)
		if err != nil {
			log.Printf("Ошибка получения items_list: %v", err)
			itemsList = []ItemsListItem{} // Пустой список в случае ошибки
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
			// Без поиска - оптимизированный запрос без image_data
			countQuery = "SELECT COUNT(*) FROM ocr_results"
			dataQuery = `SELECT ocr.id, ocr.image_path, ocr.ocr_text, ocr.debug_info, ocr.json_data, ocr.raw_text, ocr.created_at FROM ocr_results ocr ORDER BY ocr.created_at DESC LIMIT ? OFFSET ?`
		}

		// Получаем общее количество записей
		var totalCount int
		var countArgs []interface{}
		if searchQuery != "" || minPrice != "" || maxPrice != "" {
			countArgs = args
		}
		err = db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
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
			if err := rows.Scan(&res.ID, &res.ImagePath, &res.OCRText, &res.DebugInfo, &res.JSONData, &res.RawText, &res.CreatedAt); err != nil {
				continue
			}
			results = append(results, res)
		}

		// Загружаем структурированные данные одним запросом для всех результатов
		if len(results) > 0 {
			var resultIDs []string
			for _, res := range results {
				resultIDs = append(resultIDs, strconv.Itoa(res.ID))
			}

			itemsQuery := fmt.Sprintf(`SELECT id, ocr_result_id, title, title_short, enhancement, price, package, owner, count, category, created_at 
				FROM structured_items 
				WHERE ocr_result_id IN (%s) 
				ORDER BY ocr_result_id, created_at`, strings.Join(resultIDs, ","))

			itemRows, err := db.Query(itemsQuery)
			if err == nil {
				defer itemRows.Close()

				// Группируем items по ocr_result_id
				itemsByOCRID := make(map[int][]StructuredItem)
				for itemRows.Next() {
					var item StructuredItem
					if err := itemRows.Scan(&item.ID, &item.OCRResultID, &item.Title, &item.TitleShort, &item.Enhancement, &item.Price, &item.Package, &item.Owner, &item.Count, &item.Category, &item.CreatedAt); err == nil {
						itemsByOCRID[item.OCRResultID] = append(itemsByOCRID[item.OCRResultID], item)
					}
				}

				// Присваиваем items к соответствующим результатам
				for i := range results {
					results[i].Items = itemsByOCRID[results[i].ID]
				}
			}
		}

		// Получаем статус и действия
		status, err := getCurrentStatus(db)
		if err != nil {
			log.Printf("Ошибка получения статуса: %v", err)
			status = Status{ID: 0, CurrentStatus: "unknown", UpdatedAt: ""}
		}

		recentActions, err := getRecentActions(db, 5)
		if err != nil {
			log.Printf("Ошибка получения действий: %v", err)
			recentActions = []Action{}
		}

		// Подготавливаем данные для шаблона
		pageData := PageData{
			Results:       results,
			CurrentPage:   page,
			TotalPages:    totalPages,
			TotalCount:    totalCount,
			HasPrev:       page > 1,
			HasNext:       page < totalPages,
			PrevPage:      page - 1,
			NextPage:      page + 1,
			SearchQuery:   searchQuery,
			MinPrice:      minPrice,
			MaxPrice:      maxPrice,
			ActiveTab:     activeTab,
			ItemsList:     itemsList,
			Status:        status,
			RecentActions: recentActions,
		}

		renderTemplate(w, pageData)
	})

	// Обработчик для кнопки Start
	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		// Помечаем последнее невыполненное действие как выполненное
		err := updateLatestPendingAction(db)
		if err != nil {
			log.Printf("Ошибка обновления последнего действия: %v", err)
		}

		err = addActionWithExecuted(db, "start", false)
		if err != nil {
			log.Printf("Ошибка добавления действия start: %v", err)
			http.Error(w, "Internal server error", 500)
			return
		}

		// Обновляем статус на start
		err = updateStatus(db, "start")
		if err != nil {
			log.Printf("Ошибка обновления статуса: %v", err)
		}

		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	// Обработчик для кнопки Stop
	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		// Помечаем последнее невыполненное действие как выполненное
		err := updateLatestPendingAction(db)
		if err != nil {
			log.Printf("Ошибка обновления последнего действия: %v", err)
		}

		err = addActionWithExecuted(db, "stop", false)
		if err != nil {
			log.Printf("Ошибка добавления действия stop: %v", err)
			http.Error(w, "Internal server error", 500)
			return
		}

		// Обновляем статус на stop
		err = updateStatus(db, "stop")
		if err != nil {
			log.Printf("Ошибка обновления статуса: %v", err)
		}

		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	// Обработчик для кнопки Restart
	http.HandleFunc("/restart", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		// Помечаем последнее невыполненное действие как выполненное
		err := updateLatestPendingAction(db)
		if err != nil {
			log.Printf("Ошибка обновления последнего действия: %v", err)
		}

		err = addActionWithExecuted(db, "restart", false)
		if err != nil {
			log.Printf("Ошибка добавления действия restart: %v", err)
			http.Error(w, "Internal server error", 500)
			return
		}

		// Обновляем статус на restart
		err = updateStatus(db, "restart")
		if err != nil {
			log.Printf("Ошибка обновления статуса: %v", err)
		}

		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	// Обработчик для получения статуса в формате JSON
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		// Получаем текущий статус
		status, err := getCurrentStatus(db)
		if err != nil {
			log.Printf("Ошибка получения статуса: %v", err)
			http.Error(w, "Internal server error", 500)
			return
		}

		// Устанавливаем заголовки для JSON
		w.Header().Set("Content-Type", "application/json")

		// Формируем JSON ответ
		response := map[string]interface{}{
			"status":    status.CurrentStatus,
			"updatedAt": status.UpdatedAt,
		}

		// Кодируем в JSON
		jsonData, err := json.Marshal(response)
		if err != nil {
			log.Printf("Ошибка кодирования JSON: %v", err)
			http.Error(w, "Internal server error", 500)
			return
		}

		w.Write(jsonData)
	})

	// Endpoint для Prometheus метрик
	http.Handle("/metrics", promhttp.Handler())

	// Prometheus API endpoints для совместимости с Grafana
	http.HandleFunc("/api/v1/status/buildinfo", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📊 Запрос к /api/v1/status/buildinfo от %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","data":{"version":"1.0.0","revision":"","branch":"","buildUser":"","buildDate":"","goVersion":"go1.22"}}`))
	})

	http.HandleFunc("/api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, must-revalidate")

		re := regexp.MustCompile(`^([a-zA-Z0-9_]+)(\{.*\})?$`)
		matches := re.FindStringSubmatch(query)
		if len(matches) > 0 {
			metricName := matches[1]
			if metricName == "gold_coin_avg_min_3_prices" {
				ch := make(chan prometheus.Metric)
				go func() {
					goldCoinAvgPrice.Collect(ch)
					close(ch)
				}()
				var results []string
				for metric := range ch {
					dto := &io_prometheus_client.Metric{}
					if err := metric.Write(dto); err != nil {
						continue
					}
					category := ""
					for _, label := range dto.Label {
						if label.GetName() == "category" {
							category = label.GetValue()
						}
					}
					value := dto.GetGauge().GetValue()
					results = append(results, fmt.Sprintf(`{"metric":{"__name__":"gold_coin_avg_min_3_prices","category":"%s"},"value":[%d,"%f"]}`, category, int(time.Now().Unix()), value))
				}
				response := fmt.Sprintf(`{"status":"success","data":{"resultType":"vector","result":[%s]}}`, strings.Join(results, ","))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(response))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	})

	http.HandleFunc("/api/v1/query_range", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, must-revalidate")

		re := regexp.MustCompile(`^([a-zA-Z0-9_]+)(\{.*\})?$`)
		matches := re.FindStringSubmatch(query)
		if len(matches) > 0 {
			metricName := matches[1]
			if metricName == "gold_coin_avg_min_3_prices" {
				ch := make(chan prometheus.Metric)
				go func() {
					goldCoinAvgPrice.Collect(ch)
					close(ch)
				}()
				var results []string
				for metric := range ch {
					dto := &io_prometheus_client.Metric{}
					if err := metric.Write(dto); err != nil {
						continue
					}
					category := ""
					for _, label := range dto.Label {
						if label.GetName() == "category" {
							category = label.GetValue()
						}
					}
					value := dto.GetGauge().GetValue()
					// Для примера возвращаем 3 точки с разными timestamp
					values := fmt.Sprintf("[[%d,\"%f\"],[%d,\"%f\"],[%d,\"%f\"]]", int(time.Now().Unix())-30, value, int(time.Now().Unix())-15, value, int(time.Now().Unix()), value)
					results = append(results, fmt.Sprintf(`{"metric":{"__name__":"gold_coin_avg_min_3_prices","category":"%s"},"values":%s}`, category, values))
				}
				response := fmt.Sprintf(`{"status":"success","data":{"resultType":"matrix","result":[%s]}}`, strings.Join(results, ","))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(response))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
	})

	http.HandleFunc("/api/v1/series", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📊 Запрос к /api/v1/series от %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","data":[{"__name__":"gold_coin_avg_min_3_prices","category":"buy_consumables"}]}`))
	})

	// /api/v1/labels — список всех лейблов
	http.HandleFunc("/api/v1/labels", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		labels := []string{"__name__", "category"}
		resp := map[string]interface{}{
			"status": "success",
			"data":   labels,
		}
		json.NewEncoder(w).Encode(resp)
	})

	// /api/v1/label/<label_name>/values — значения для конкретного лейбла
	http.HandleFunc("/api/v1/label/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 6 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":"error","errorType":"bad_data","error":"label name required"}`))
			return
		}
		labelName := parts[5]
		valuesSet := make(map[string]struct{})
		ch := make(chan prometheus.Metric)
		go func() {
			goldCoinAvgPrice.Collect(ch)
			close(ch)
		}()
		for metric := range ch {
			dto := &io_prometheus_client.Metric{}
			if err := metric.Write(dto); err != nil {
				continue
			}
			for _, label := range dto.Label {
				if label.GetName() == labelName {
					valuesSet[label.GetValue()] = struct{}{}
				}
			}
		}
		var values []string
		for v := range valuesSet {
			values = append(values, v)
		}
		resp := map[string]interface{}{
			"status": "success",
			"data":   values,
		}
		json.NewEncoder(w).Encode(resp)
	})

	// /api/v1/metadata — возвращает метаданные о метриках
	http.HandleFunc("/api/v1/metadata", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		metadata := map[string][]map[string]string{
			"gold_coin_avg_min_3_prices": {{
				"type": "gauge",
				"help": "Среднее из 3 минимальных цен для gold coin",
				"unit": "",
			}},
			"gold_coin_min_price": {{
				"type": "gauge",
				"help": "Минимальная цена для gold coin",
				"unit": "",
			}},
			"gold_coin_max_price_of_min_3": {{
				"type": "gauge",
				"help": "Максимальная из 3 минимальных цен для gold coin",
				"unit": "",
			}},
			"gold_coin_prices_count": {{
				"type": "gauge",
				"help": "Количество цен для gold coin",
				"unit": "",
			}},
			"shnyr_test_metric": {{
				"type": "gauge",
				"help": "Тестовая метрика для проверки работы Prometheus",
				"unit": "",
			}},
		}
		resp := map[string]interface{}{
			"status": "success",
			"data":   metadata,
		}
		json.NewEncoder(w).Encode(resp)
	})

	// Простой health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status": "ok", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// Запускаем периодическое обновление метрик
	go func() {
		for {
			updateGoldCoinMetrics(db)
			time.Sleep(30 * time.Second) // Обновляем каждые 30 секунд
		}
	}()

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
			return strings.ReplaceAll(strings.ReplaceAll(s, `\\`, `\\\\`), `\"`, `\\\"`)
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
		"formatStatus": func(status string) string {
			switch status {
			case "stopped":
				return "🔴 СТРАДАЕТ ХУЙНЕЙ"
			case "main":
				return "🟢 ОХОТА НА ЛОХА: Запуск приложения"
			case "ready":
				return "🟢 ОХОТА НА ЛОХА: Готов к работе"
			case "cycle_all_items":
				return "🟢 ОХОТА НА ЛОХА: cycle_all_items"
			case "cycle_listed_items":
				return "🟢 ОХОТА НА ЛОХА: cycle_listed_items"
			case "running":
				return "🟢 ОХОТА НА ЛОХА"
			case "paused":
				return "🟡 ОХОТА НА ЛОХА: Приостановлено"
			case "error":
				return "❌ ОХОТА НА ЛОХА: Ошибка"
			case "unknown":
				return "❓ ОХОТА НА ЛОХА: Неизвестно"
			default:
				// Если статус содержит название скрипта, форматируем его
				if strings.Contains(status, "cycle_") || strings.Contains(status, "ocr_") || strings.Contains(status, "web_") {
					return "🟢 ОХОТА НА ЛОХА: " + status
				}
				return "🟢 ОХОТА НА ЛОХА: " + status
			}
		},
		"int": func(x float64) int { return int(x) },
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
