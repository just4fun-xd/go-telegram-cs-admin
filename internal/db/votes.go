package db

import (
	"fmt"
	"strings"
)

// CountVotesForDate — количество голосов за конкретную дату
func CountVotesForDate(pollID, voteDate string) int {
	var count int64
	DB.Model(&Vote{}).
		Where("poll_id = ? AND vote_date = ?", pollID, voteDate).
		Count(&count)
	return int(count)
}

// GetVotersForDate — список пользователей, проголосовавших за дату
func GetVotersForDate(pollID, voteDate string) string {
	var votes []Vote
	DB.Where("poll_id = ? AND vote_date = ?", pollID, voteDate).Find(&votes)

	var sb strings.Builder
	for _, v := range votes {
		// Если userName пуст, можно показывать ID или "Аноним"
		line := fmt.Sprintf("@%s\n", v.UserName)
		sb.WriteString(line)
	}
	return sb.String()
}
