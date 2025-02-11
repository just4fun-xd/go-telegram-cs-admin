package bot

import (
	"log"
	"time"

	"go-telegram-cs-admin/internal/db"
)

// StartCleanupRoutine — пример очистки каждую субботу (или другую логику)
func StartCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			now := time.Now()
			// Каждую субботу в 14:00, например
			if now.Weekday() == time.Saturday && now.Hour() == 14 && now.Minute() == 0 {
				log.Println("🧹 Очистка старых опросов...")
				count, err := CleanupOldPolls()
				if err != nil {
					log.Println("Ошибка очистки:", err)
				} else {
					log.Printf("✅ Удалено %d опросов", count)
				}
			}
		}
	}()
}

// CleanupOldPolls — удаляет все опросы, дата встречи которых прошла
func CleanupOldPolls() (int, error) {
	now := time.Now()
	var polls []db.Poll
	if err := db.DB.Where("event_date < ?", now).Find(&polls).Error; err != nil {
		return 0, err
	}

	count := 0
	for _, p := range polls {
		if err := db.DB.Where("poll_id = ?", p.PollID).Unscoped().Delete(&db.Vote{}).Error; err != nil {
			return count, err
		}
		if err := db.DB.Unscoped().Delete(&p).Error; err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
