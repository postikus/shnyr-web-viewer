# Динамический поиск окна игры

## Описание

Добавлена функциональность автоматического поиска окна игры на экране по цвету пикселей. Это позволяет программе работать независимо от положения окна игры на экране.

## Как это работает

1. **Захват всего экрана** - программа делает скриншот всего экрана
2. **Поиск окна игры** - ищет область размером примерно 400x500 пикселей, где есть нечерные пиксели (RGB >= 10)
3. **Расчет координат** - все координаты из конфига теперь интерпретируются как относительные к найденному окну

## Новые файлы

- `internal/types/game_window.go` - тип для представления найденного окна
- `internal/image/window_finder.go` - алгоритм поиска окна по цвету
- `internal/screen/screen.go` - функции для работы с экраном
- `internal/config/dynamic_config.go` - динамическая конфигурация
- `examples/dynamic_example.go` - пример использования

## Использование

### Базовое использование

```go
// Создаем обычную конфигурацию
err, c := config.InitConfig()

// Создаем динамическую конфигурацию
dynamicConfig := config.NewDynamicConfig(&c)

// Находим окно игры
err = dynamicConfig.FindAndSetGameWindow()
if err != nil {
    log.Printf("Warning: Could not find game window, using static coordinates: %v", err)
} else {
    log.Printf("Game window found at: X=%d, Y=%d, Width=%d, Height=%d", 
        dynamicConfig.GameWindow.X, dynamicConfig.GameWindow.Y, 
        dynamicConfig.GameWindow.Width, dynamicConfig.GameWindow.Height)
}
```

### Получение абсолютных координат

```go
// Для кликов
absX, absY := dynamicConfig.GetAbsoluteCoordinates(c.Click.Button1)

// Для скриншотов
absScreenshot := dynamicConfig.GetAbsoluteCoordinatesWithSize(c.Screenshot.ItemList)
```

## Параметры поиска

- **Размер окна**: примерно 400x500 пикселей
- **Цвет фона**: черный (RGB < 10)
- **Порог нечерных пикселей**: минимум 5% от общей площади окна

## Совместимость

Если окно игры не найдено, программа автоматически использует статические координаты из конфига. Это обеспечивает обратную совместимость.

## Запуск примера

```bash
go run examples/dynamic_example.go
```

## Настройка

Для изменения параметров поиска отредактируйте функции в `internal/image/window_finder.go`:

- `targetWidth` и `targetHeight` - ожидаемый размер окна
- `threshold` - порог нечерных пикселей
- `step` - шаг поиска (влияет на скорость) 