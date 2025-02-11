package bot

import (
	"fmt"
	"go-telegram-cs-admin/internal/db"
	"log"
)

// replaceParticipant переносит все голоса @oldUser → @newUser
func replaceParticipant(oldUser, newUser string) error {
	var oldVotes []db.Vote
	if err := db.DB.Where("user_name = ?", oldUser).Find(&oldVotes).Error; err != nil {
		return err
	}
	if len(oldVotes) == 0 {
		return fmt.Errorf("не найден старый пользователь @%s", oldUser)
	}
	for _, v := range oldVotes {
		newVote := db.Vote{
			PollID:   v.PollID,
			UserID:   v.UserID,
			UserName: newUser,
			ChatID:   v.ChatID,
			VoteDate: v.VoteDate,
		}
		if err := db.DB.Create(&newVote).Error; err != nil {
			return err
		}
	}
	if err := db.DB.Where("user_name = ?", oldUser).Delete(&db.Vote{}).Error; err != nil {
		return err
	}
	log.Printf("✅ Заменён пользователь @%s -> @%s", oldUser, newUser)
	return nil
}
