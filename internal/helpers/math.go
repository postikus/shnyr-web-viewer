package helpers

// Min возвращает минимальное из двух чисел
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
