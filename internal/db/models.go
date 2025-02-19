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
	OptionsCount int
	EventDate    time.Time
	ReminderDate *time.Time
	Reminded     bool
	PollDay      string
}

type Vote struct {
	gorm.Model
	PollID   string
	UserID   int64
	UserName string
	ChatID   int64
	VoteDate string
}

type Reminder struct {
	gorm.Model
	PollID       string    // связывает с Poll
	OptionDate   string    // "Понедельник (17.02)" или "Вторник (18.02)" и т.д.
	ReminderTime time.Time // когда нужно отправить напоминание
	Reminded     bool      // было ли уже отправлено напоминание
}
