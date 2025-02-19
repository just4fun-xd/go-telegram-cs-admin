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
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	err = DB.AutoMigrate(&Poll{}, &Vote{}, &Reminder{})
	if err != nil {
		log.Fatalf("Ошибка миграции таблиц: %v", err)
	}

	log.Println("✅ База данных инициализирована")
}
