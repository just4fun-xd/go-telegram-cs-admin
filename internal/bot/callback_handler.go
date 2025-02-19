package bot

import (
	"fmt"
	"go-telegram-cs-admin/internal/db"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleCallbackQuery(bot *tgbotapi.BotAPI, cq *tgbotapi.CallbackQuery) {
	data := cq.Data

	switch {
	case strings.HasPrefix(data, "close_"):
		msgIDStr := strings.TrimPrefix(data, "close_")
		handleClosePollInline(bot, cq, strToInt(msgIDStr))

	case strings.HasPrefix(data, "delete_"):
		msgIDStr := strings.TrimPrefix(data, "delete_")
		handleDeletePollInline(bot, cq, strToInt(msgIDStr))

	case strings.HasPrefix(data, "rd|"):
		shortID := strings.TrimPrefix(data, "rd|")
		handleReplaceDayShortID(bot, cq, shortID)

	default:
		// не обрабатываем
	}

	// Убираем "спиннер"
	bot.Request(tgbotapi.NewCallback(cq.ID, ""))
}

func handleClosePollInline(bot *tgbotapi.BotAPI, cq *tgbotapi.CallbackQuery, messageID int) {
	chatID := cq.Message.Chat.ID

	poll, err := findPollByChatMessageID(chatID, messageID)
	if err != nil {
		bot.Request(tgbotapi.NewCallbackWithAlert(cq.ID, "Опрос не найден"))
		log.Printf("Опрос не найден: %v", err)
		return
	}
	if poll.IsClosed {
		bot.Request(tgbotapi.NewCallbackWithAlert(cq.ID, "Опрос уже закрыт!"))
		return
	}

	stopCfg := tgbotapi.NewStopPoll(chatID, messageID)
	if _, err := bot.StopPoll(stopCfg); err != nil {
		bot.Request(tgbotapi.NewCallbackWithAlert(cq.ID, fmt.Sprintf("Ошибка: %v", err)))
		return
	}
	poll.IsClosed = true
	savePoll(poll)

	// Заменяем клавиатуру, оставляя кнопку "Удалить"
	newMarkup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалить", fmt.Sprintf("delete_%d", messageID)),
		),
	)
	editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, newMarkup)
	bot.Send(editMsg)

	bot.Request(tgbotapi.NewCallback(cq.ID, "Опрос закрыт!"))
	log.Printf("🛑 Опрос PollID=%s закрыт, msgID=%d", poll.PollID, messageID)
}

func handleDeletePollInline(bot *tgbotapi.BotAPI, cq *tgbotapi.CallbackQuery, messageID int) {
	chatID := cq.Message.Chat.ID

	delMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	if _, err := bot.Request(delMsg); err != nil {
		bot.Request(tgbotapi.NewCallbackWithAlert(cq.ID, fmt.Sprintf("Ошибка удаления: %v", err)))
		return
	}

	bot.Request(tgbotapi.NewCallback(cq.ID, "Сообщение удалено!"))
	log.Printf("✅ Сообщение msgID=%d удалено", messageID)
}

func handleReplaceDayShortID(bot *tgbotapi.BotAPI, cq *tgbotapi.CallbackQuery, shortID string) {
	val, ok := replaceDayCache.Load(shortID)
	if !ok {
		bot.Request(tgbotapi.NewCallbackWithAlert(cq.ID, "Данные не найдены"))
		return
	}
	ctx, ok := val.(*ReplaceDayContext)
	if !ok {
		bot.Request(tgbotapi.NewCallbackWithAlert(cq.ID, "Ошибка контекста"))
		return
	}
	replaceDayCache.Delete(shortID)

	// Выполняем замену
	if err := replaceParticipantWeekly(ctx.PollID, ctx.OldUser, ctx.NewUser, ctx.Day); err != nil {
		bot.Request(tgbotapi.NewCallbackWithAlert(cq.ID, fmt.Sprintf("Ошибка: %v", err)))
		return
	}

	// Удаляем сообщение. НЕ cq.Message.MessageID, а именно ctx.KeyboardMsgID / ctx.KeyboardChatID
	if ctx.KeyboardMsgID != 0 && ctx.KeyboardChatID != 0 {
		delMsg := tgbotapi.NewDeleteMessage(ctx.KeyboardChatID, ctx.KeyboardMsgID)
		if _, err := bot.Request(delMsg); err != nil {
			log.Printf("Ошибка удаления сообщения клавиатуры: %v", err)
		}
	} else {
		log.Println("Предупреждение: ctx.KeyboardMsgID=0, возможно ID не сохранён.")
	}

	// Отправляем новое сообщение
	success := tgbotapi.NewMessage(cq.Message.Chat.ID,
		fmt.Sprintf("✅ Участник @%s заменён на @%s в день '%s'.", ctx.OldUser, ctx.NewUser, ctx.Day),
	)
	if _, err := bot.Send(success); err != nil {
		log.Printf("Ошибка отправки итогового сообщения: %v", err)
	}

	log.Printf("✅ Заменён @%s -> @%s (Day=%s, PollID=%s)", ctx.OldUser, ctx.NewUser, ctx.Day, ctx.PollID)
	// bot.Request(tgbotapi.NewCallback(cq.ID, "Участник заменён!"))
}

func findPollByChatMessageID(chatID int64, messageID int) (*db.Poll, error) {
	var p db.Poll
	err := db.DB.Where("chat_id = ? AND message_id = ?", chatID, messageID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func strToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
