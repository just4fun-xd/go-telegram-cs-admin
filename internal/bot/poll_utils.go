package bot

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"

	"go-telegram-cs-admin/internal/db"
	"go-telegram-cs-admin/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ReplaceDayContext хранит данные для операции "заменить oldUser на newUser"
type ReplaceDayContext struct {
	PollID         string
	OldUser        string
	NewUser        string
	Day            string
	KeyboardMsgID  int
	KeyboardChatID int64
}

var replaceDayCache sync.Map

func generateShortID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// replaceParticipantDay ...
func replaceParticipantDay(p *db.Poll, oldUser, newUser string) error {
	if err := db.DB.Where("poll_id = ? AND user_name = ? AND vote_date = ?", p.PollID, oldUser, p.PollDay).
		Delete(&db.Vote{}).Error; err != nil {
		return err
	}
	return db.DB.Create(&db.Vote{
		PollID:   p.PollID,
		UserName: newUser,
		ChatID:   p.ChatID,
		VoteDate: p.PollDay,
	}).Error
}

func replaceParticipantWeekly(pollID, oldUser, newUser, voteDate string) error {
	// Удаляем oldUser
	if err := db.DB.Where("poll_id = ? AND user_name = ? AND vote_date = ?", pollID, oldUser, voteDate).
		Delete(&db.Vote{}).Error; err != nil {
		return err
	}
	// Добавляем newUser
	return db.DB.Create(&db.Vote{
		PollID:   pollID,
		UserName: newUser,
		VoteDate: voteDate,
	}).Error
}

func findPollByMessageID(chatID int64, messageID int) (*db.Poll, error) {
	var p db.Poll
	if err := db.DB.Where("chat_id = ? AND message_id = ?", chatID, messageID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// При необходимости:
func findPollByPollID(pollID string) (*db.Poll, error) {
	var p db.Poll
	if err := db.DB.Where("poll_id = ?", pollID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func savePoll(p *db.Poll) {
	db.DB.Save(p)
}

// sendReplaceDayKeyboard — шлёт одно сообщение с кнопками выбора дня
func sendReplaceDayKeyboard(bot *tgbotapi.BotAPI, p *db.Poll, oldUser, newUser string, replyMsgID int) {
	days := utils.GeneratePollOptions()
	var rows [][]tgbotapi.InlineKeyboardButton

	// Создаём для каждого дня уникальный shortID, но пока не знаем MessageID
	for _, day := range days {
		sid := generateShortID()
		ctx := &ReplaceDayContext{
			PollID:  p.PollID,
			OldUser: oldUser,
			NewUser: newUser,
			Day:     day,
			// KeyboardMsgID / KeyboardChatID заполним после Send
		}
		replaceDayCache.Store(sid, ctx)

		cbData := fmt.Sprintf("rd|%s", sid)
		btn := tgbotapi.NewInlineKeyboardButtonData(day, cbData)
		row := tgbotapi.NewInlineKeyboardRow(btn)
		rows = append(rows, row)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(p.ChatID, "Выберите день для замены участника:")
	msg.ReplyToMessageID = replyMsgID
	msg.ReplyMarkup = keyboard

	// Отправляем сообщение
	sentMsg, err := bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки клавиатуры выбора дня: %v", err)
		return
	}
	log.Printf("Отправлено сообщение выбора дня, messageID=%d", sentMsg.MessageID)

	// Теперь обновим в кэше KeyboardMsgID / KeyboardChatID
	fixDayKeyboardContext(p.ChatID, int(sentMsg.MessageID), oldUser, newUser)
}

// fixDayKeyboardContext — обновляет поля KeyboardMsgID, KeyboardChatID
func fixDayKeyboardContext(chatID int64, messageID int, oldUser, newUser string) {
	replaceDayCache.Range(func(key, val interface{}) bool {
		sid, ok := key.(string)
		if !ok {
			return true
		}
		ctx, ok := val.(*ReplaceDayContext)
		if !ok {
			return true
		}
		// Сравним oldUser,newUser
		if ctx.OldUser == oldUser && ctx.NewUser == newUser && ctx.KeyboardMsgID == 0 {
			ctx.KeyboardMsgID = messageID
			ctx.KeyboardChatID = chatID
			replaceDayCache.Store(sid, ctx)
		}
		return true
	})
}
