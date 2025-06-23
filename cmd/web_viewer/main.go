package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

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

func main() {
	db, err := sql.Open("mysql", "root:root@tcp(108.181.194.102:3306)/octopus?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`SELECT id, image_path, image_data, ocr_text, debug_info, json_data, raw_text, created_at FROM ocr_results ORDER BY created_at DESC LIMIT 20`)
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

		tmpl := `
		<html><head><title>OCR Results</title></head><body>
		<h2>Последние результаты OCR</h2>
		<table border=1 cellpadding=5>
		<tr><th>ID</th><th>Image</th><th>Debug Info</th><th>Raw Text</th><th>JSON Data</th><th>Structured Items</th><th>Created</th></tr>
		{{range .}}
		<tr>
		<td>{{.ID}}</td>
		<td>
			{{if .ImageData}}
			<img src="data:image/png;base64,{{base64encode .ImageData}}" style="max-width:300px;max-height:200px;" />
			{{else}}
			No image data
			{{end}}
		</td>
		<td><pre style="max-width:300px;max-height:200px;overflow:auto;">{{.DebugInfo}}</pre></td>
		<td><pre style="max-width:300px;max-height:200px;overflow:auto;">{{.RawText}}</pre></td>
		<td><pre style="max-width:300px;max-height:200px;overflow:auto;">{{.JSONData}}</pre></td>
		<td>
			{{if .Items}}
			<table border=1 style="font-size:12px;">
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
			{{else}}
			No structured data
			{{end}}
		</td>
		<td>{{.CreatedAt}}</td>
		</tr>
		{{end}}
		</table>
		</body></html>`

		t, err := template.New("web").Funcs(template.FuncMap{
			"base64encode": func(data []byte) string {
				return base64.StdEncoding.EncodeToString(data)
			},
		}).Parse(tmpl)
		if err != nil {
			http.Error(w, "Template error", 500)
			return
		}
		t.Execute(w, results)
	})

	fmt.Println("Откройте http://localhost:8080 в браузере для просмотра результатов OCR")
	http.ListenAndServe(":8080", nil)
}
