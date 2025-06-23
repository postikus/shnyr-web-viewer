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

type OCRResult struct {
	ID        int
	ImagePath string
	ImageData []byte
	OCRText   string
	DebugInfo string
	JSONData  string
	CreatedAt string
}

func main() {
	db, err := sql.Open("mysql", "root:root@tcp(108.181.194.102:3306)/octopus?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`SELECT id, image_path, image_data, ocr_text, debug_info, json_data, created_at FROM ocr_results ORDER BY created_at DESC LIMIT 20`)
		if err != nil {
			http.Error(w, "DB error", 500)
			return
		}
		defer rows.Close()

		var results []OCRResult
		for rows.Next() {
			var res OCRResult
			if err := rows.Scan(&res.ID, &res.ImagePath, &res.ImageData, &res.OCRText, &res.DebugInfo, &res.JSONData, &res.CreatedAt); err != nil {
				continue
			}
			results = append(results, res)
		}

		tmpl := `
		<html><head><title>OCR Results</title></head><body>
		<h2>Последние результаты OCR</h2>
		<table border=1 cellpadding=5>
		<tr><th>ID</th><th>Image</th><th>Debug Info</th><th>JSON Data</th><th>Created</th></tr>
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
		<td><pre style="max-width:300px;max-height:200px;overflow:auto;">{{.JSONData}}</pre></td>
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
