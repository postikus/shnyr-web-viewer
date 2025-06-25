package config

import (
	"fmt"
	"image"
	"log"

	"github.com/spf13/viper"
)

// Структура для координат с размером
type CoordinatesWithSize struct {
	X      int `mapstructure:"x"`
	Y      int `mapstructure:"y"`
	Width  int `mapstructure:"width"`
	Height int `mapstructure:"height"`
}

// Структура для элементов
type Screenshot struct {
	ItemList                     CoordinatesWithSize `mapstructure:"item_list"`
	ItemOffersListWithoutButtons CoordinatesWithSize `mapstructure:"item_offers_list_without_buttons"`
	ItemOffersListWithButtons    CoordinatesWithSize `mapstructure:"item_offers_list_with_buttons"`
	ItemOffersListButtons        CoordinatesWithSize `mapstructure:"item_offers_list_buttons"`
	Item1                        CoordinatesWithSize `mapstructure:"item1"`
	Item2                        CoordinatesWithSize `mapstructure:"item2"`
}

// Структура для кликов
type Click struct {
	Back    image.Point `mapstructure:"back"`
	Button1 image.Point `mapstructure:"button1"`
	Button2 image.Point `mapstructure:"button2"`
	Button3 image.Point `mapstructure:"button3"`
	Button4 image.Point `mapstructure:"button4"`
	Button5 image.Point `mapstructure:"button5"`
	Button6 image.Point `mapstructure:"button6"`
	Item1   image.Point `mapstructure:"item1"`
	Item2   image.Point `mapstructure:"item2"`
	Item3   image.Point `mapstructure:"item3"`
	Item4   image.Point `mapstructure:"item4"`
	Item5   image.Point `mapstructure:"item5"`
	Item6   image.Point `mapstructure:"item6"`
	Item7   image.Point `mapstructure:"item7"`
	Item8   image.Point `mapstructure:"item8"`
	Item9   image.Point `mapstructure:"item9"`
	Scroll  image.Point `mapstructure:"scroll"`
}

// Основная структура конфигурации
type Config struct {
	Port                        string     `mapstructure:"port"`
	BaudRate                    int        `mapstructure:"baud_rate"`
	WindowTopOffset             int        `mapstructure:"window_top_offset"`
	ListButtonBottomYCoordinate int        `mapstructure:"list_button_bottom_y_coordinate"`
	MaxCyclesItemsList          int        `mapstructure:"max_cycles_items_list"`
	LogFilePath                 string     `mapstructure:"log_file_path"`
	Screenshot                  Screenshot `mapstructure:"screenshot"`
	Click                       Click      `mapstructure:"click"`
	SaveToDB                    int        `mapstructure:"save_to_db"`
}

var InitConfig = func() (error, Config) {
	// Инициализация viper для чтения конфигурации из .yaml файла
	viper.SetConfigName("config") // Имя конфигурационного файла без расширения
	viper.AddConfigPath(".")      // Путь к файлу конфигурации
	viper.SetConfigType("yaml")   // Формат файла

	// Чтение конфигурации
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	// Выводим все конфигурации для диагностики
	fmt.Println("Config loaded:")
	fmt.Println(viper.AllSettings()) // Вывод всех настроек

	// Создание структуры и заполнение её данными из конфигурации
	var config Config
	err := viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
		return err, config
	}

	return nil, config
}
