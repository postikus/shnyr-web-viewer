package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: go run . <example_name>")
		fmt.Println("Доступные примеры:")
		fmt.Println("  dynamic - динамический пример")
		fmt.Println("  find_items - поиск предметов")
		fmt.Println("  find_window - поиск окна")
		fmt.Println("  stripe_analysis - анализ полосок")
		fmt.Println("  stripe_diff - разница полосок")
		return
	}

	exampleName := os.Args[1]

	switch exampleName {
	case "find_items":
		findItemsMain()
	case "find_window":
		findWindowMain()
	case "stripe_analysis":
		stripeAnalysisMain()
	case "stripe_diff":
		stripeDiffMain()
	default:
		fmt.Printf("Неизвестный пример: %s\n", exampleName)
	}
}
