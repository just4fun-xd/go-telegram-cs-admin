package bot

import (
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"go-telegram-cs-admin/internal/constants" // Пример: меняйте под свою структуру
	"go-telegram-cs-admin/internal/db"
)

var lastQuietMessage = make(map[int64]int)

// хранит MessageID последнего "тихого" сообщения (help/unknown) по каждому chatID

// HandleMessage обрабатывает все входящие TEXT-сообщения (не poll_answer) из update.Message
func HandleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	// 1. Если это команда (msg.IsCommand() == true)
	if msg.IsCommand() {
		// Удаляем предыдущие "тихие" сообщения (help или unknown), если есть
		if oldMsgID, ok := lastQuietMessage[chatID]; ok {
			bot.Request(tgbotapi.NewDeleteMessage(chatID, oldMsgID))
			delete(lastQuietMessage, chatID)
		}

		switch msg.Command() {
		case "start":
			sendNormalMessage(bot, chatID, constants.MsgStart)

		case "help":
			// Отправляем help в тихом режиме
			sendQuietMessage(bot, chatID, constants.MsgHelp)

		case "poll":
			// Создать опрос
			SendPoll(bot, chatID)
			// См. предыдущие файлы poll.go

		case "replace":
			// /replace @OldUser @NewUser
			args := strings.Split(msg.Text, " ")
			if len(args) < 3 {
				sendQuietMessage(bot, chatID, constants.MsgReplaceFormat)
				return
			}
			oldUser := strings.TrimPrefix(args[1], "@")
			newUser := strings.TrimPrefix(args[2], "@")

			if err := replaceParticipant(oldUser, newUser); err != nil {
				sendNormalMessage(bot, chatID, "Ошибка при замене: "+err.Error())
			} else {
				sendNormalMessage(bot, chatID, "✅ Участник @"+oldUser+" заменён на @"+newUser)
			}

		case "close_poll":
			// Команда закрытия опроса
			if msg.ReplyToMessage == nil {
				sendQuietMessage(bot, chatID, constants.MsgReplyClose)
				return
			}
			// Узнаём pollID из БД по ReplyToMessage.MessageID
			pollID, err := getPollIDByMessageID(chatID, msg.ReplyToMessage.MessageID)
			if err != nil {
				sendNormalMessage(bot, chatID, "Не удалось определить PollID: "+err.Error())
				return
			}
			if err := ClosePoll(bot, pollID); err != nil {
				sendNormalMessage(bot, chatID, "Ошибка закрытия опроса: "+err.Error())
			} else {
				sendNormalMessage(bot, chatID, constants.MsgPollClosed)
			}
		case "players":
			// Новая команда: вывести список участников
			if msg.ReplyToMessage == nil {
				// Пользователь не ответил на сообщение с опросом
				sendQuietMessage(bot, chatID, constants.MsgPlayersReply)
				return
			}
			pollID, err := getPollIDByMessageID(chatID, msg.ReplyToMessage.MessageID)
			if err != nil {
				sendNormalMessage(bot, chatID, "Не удалось определить PollID: "+err.Error())
				return
			}
			// С помощью pollID достаём всех участников из БД
			playersList := getAllVotersForPoll(pollID)
			if playersList == "" {
				sendNormalMessage(bot, chatID, "В этом опросе пока никто не голосовал.")
			} else {
				sendNormalMessage(bot, chatID, "Список участников:\n"+playersList)
			}

		default:
			// Если команда неизвестна
			sendQuietMessage(bot, chatID, constants.MsgUnknownCommand)
		}

		// 2. Если это НЕ команда (обычный текст)
	} else {
		// Удаляем предыдущее "тихое" сообщение (help или unknown), если есть
		if oldMsgID, ok := lastQuietMessage[chatID]; ok {
			bot.Request(tgbotapi.NewDeleteMessage(chatID, oldMsgID))
			delete(lastQuietMessage, chatID)
		}

		// Отправляем новое "тихое" сообщение
		sendQuietMessage(bot, chatID, constants.MsgUnknownCommand)
	}
}

// ----------------- ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ -----------------

// sendNormalMessage — обычное сообщение со звуком
func sendNormalMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	// Можно ещё настроить DisableNotification=false (по умолчанию)
	msg.DisableNotification = false
	_, err := bot.Send(msg)
	if err != nil {
		log.Println("Ошибка отправки сообщения:", err)
	}
}

// sendQuietMessage — тихое сообщение (DisableNotification=true) + сохраним ID
func sendQuietMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.DisableNotification = true // тихий режим
	sent, err := bot.Send(msg)
	if err != nil {
		log.Println("Ошибка отправки тихого сообщения:", err)
		return
	}
	lastQuietMessage[chatID] = sent.MessageID
}

// getPollIDByMessageID — ищет опрос (Poll) в БД по chatID и messageID
func getPollIDByMessageID(chatID int64, messageID int) (string, error) {
	var p db.Poll
	err := db.DB.Where("chat_id = ? AND message_id = ?", chatID, messageID).First(&p).Error
	if err != nil {
		return "", err
	}
	return p.PollID, nil
}
