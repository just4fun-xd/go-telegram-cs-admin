package db

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("bot.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}

	// Создаём/обновляем схемы
	err = DB.AutoMigrate(&Poll{}, &Vote{})
	if err != nil {
		log.Fatal("Ошибка миграции:", err)
	}

	log.Println("✅ База данных инициализирована")
}
