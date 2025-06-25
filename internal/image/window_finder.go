package image

import (
	"fmt"
	"image"
)

// GameWindow представляет найденное окно игры
type GameWindow struct {
	X, Y, Width, Height int
}

// FindGameWindow ищет первую нечерную точку, затем расширяет прямоугольник до границ окна (граница — черный цвет)
func FindGameWindow(img image.Image) (*GameWindow, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 1. Найти первую нечерную точку
	found := false
	var startX, startY int
	for y := 0; y < height && !found; y++ {
		for x := 0; x < width && !found; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8 := r >> 8
			g8 := g >> 8
			b8 := b >> 8
			if r8 >= 10 || g8 >= 10 || b8 >= 10 {
				startX, startY = x, y
				found = true
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("game window not found")
	}

	// 2. Расширяем прямоугольник до границ окна (граница — черный цвет)
	left, right := startX, startX
	top, bottom := startY, startY

	// Вправо
	for x := startX; x < width; x++ {
		r, g, b, _ := img.At(x, startY).RGBA()
		r8 := r >> 8
		g8 := g >> 8
		b8 := b >> 8
		if r8 < 10 && g8 < 10 && b8 < 10 {
			break
		}
		right = x
	}
	// Влево
	for x := startX; x >= 0; x-- {
		r, g, b, _ := img.At(x, startY).RGBA()
		r8 := r >> 8
		g8 := g >> 8
		b8 := b >> 8
		if r8 < 10 && g8 < 10 && b8 < 10 {
			break
		}
		left = x
	}
	// Вниз
	for y := startY; y < height; y++ {
		r, g, b, _ := img.At(startX, y).RGBA()
		r8 := r >> 8
		g8 := g >> 8
		b8 := b >> 8
		if r8 < 10 && g8 < 10 && b8 < 10 {
			break
		}
		bottom = y
	}
	// Вверх
	for y := startY; y >= 0; y-- {
		r, g, b, _ := img.At(startX, y).RGBA()
		r8 := r >> 8
		g8 := g >> 8
		b8 := b >> 8
		if r8 < 10 && g8 < 10 && b8 < 10 {
			break
		}
		top = y
	}

	return &GameWindow{
		X:      left,
		Y:      top,
		Width:  right - left + 1,
		Height: bottom - top + 1,
	}, nil
}

// ConvertToAbsoluteCoordinates конвертирует относительные координаты в абсолютные
func ConvertToAbsoluteCoordinates(window *GameWindow, relX, relY int) (int, int) {
	return window.X + relX, window.Y + relY
}
