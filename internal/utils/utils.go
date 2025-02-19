package utils

import (
	"fmt"
	"regexp"
	"time"
)

var DayRegex = regexp.MustCompile(`\((\d{2})\.(\d{2})\)`)

func GeneratePollOptions() []string {
	now := time.Now()
	nextMonday := now.AddDate(0, 0, int(time.Monday)-int(now.Weekday())+7)
	days := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница"}
	options := make([]string, 0, len(days))

	for i, day := range days {
		date := nextMonday.AddDate(0, 0, i)
		options = append(options, fmt.Sprintf("%s (%02d.%02d)", day, date.Day(), date.Month()))
	}

	return options
}
