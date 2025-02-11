package db

import (
	"time"

	"gorm.io/gorm"
)

type Poll struct {
	gorm.Model
	PollID       string `gorm:"uniqueIndex"`
	ChatID       int64
	MessageID    int
	IsClosed     bool
	EventDate    time.Time  // Дата (и время) встречи
	ReminderDate *time.Time // Время, когда нужно отправить напоминание
	Reminded     bool       // Отправлено ли напоминание
}

type Vote struct {
	gorm.Model
	PollID   string
	UserID   int64
	UserName string
	ChatID   int64
	VoteDate string
}
