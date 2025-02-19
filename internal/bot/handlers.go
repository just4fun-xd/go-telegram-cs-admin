package bot

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"go-telegram-cs-admin/internal/constants"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var lastQuietMessage = make(map[int64]int)

func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		HandleCallbackQuery(bot, update.CallbackQuery)
		return
	}
	if update.Message != nil {
		HandleMessage(bot, update.Message)
	}
}

func sanitizeText(text string) string {
	re := regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	return re.ReplaceAllString(text, "")
}

func HandleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	defer deleteUserCommand(bot, msg)

	if msg.IsCommand() {
		if oldMsgID, ok := lastQuietMessage[chatID]; ok {
			bot.Request(tgbotapi.NewDeleteMessage(chatID, oldMsgID))
			delete(lastQuietMessage, chatID)
		}

		switch msg.Command() {

		case "start":
			sendNormalMessage(bot, chatID, constants.MsgStart)

		case "help":
			sendQuietMessage(bot, chatID, constants.MsgHelp)

		case "poll":
			SendPoll(bot, chatID)

		case "poll_day":
			args := strings.Split(msg.Text, " ")
			if len(args) < 2 {
				sendQuietMessage(bot, chatID, "Формат: /poll_day DD.MM")
				return
			}
			dayStr := args[1]
			match, _ := regexp.MatchString(`^\d{2}\.\d{2}$`, dayStr)
			if !match {
				sendQuietMessage(bot, chatID, "Неверный формат даты (DD.MM). Пример: 23.02")
				return
			}
			SendDayPoll(bot, chatID, dayStr)

		case "cleanup":
			count, err := CleanupOldPolls()
			if err != nil {
				sendNormalMessage(bot, chatID, fmt.Sprintf("Ошибка очистки: %v", err))
			} else {
				sendNormalMessage(bot, chatID, fmt.Sprintf("Удалено %d старых опрос(ов)", count))
			}

		case "replace":
			if msg.ReplyToMessage == nil {
				sendQuietMessage(bot, chatID, "Команда /replace должна быть ответом на сообщение (опрос или список участников).")
				return
			}
			args := strings.Split(msg.Text, " ")
			if len(args) < 3 {
				sendQuietMessage(bot, chatID, "Формат: /replace @oldUser @newUser")
				return
			}
			oldUser := strings.TrimPrefix(args[1], "@")
			newUser := strings.TrimPrefix(args[2], "@")

			pollObj, err := findPollByMessageID(chatID, msg.ReplyToMessage.MessageID)
			if err != nil {
				sendQuietMessage(bot, chatID, fmt.Sprintf("Не удалось найти опрос: %v", err))
				return
			}

			if pollObj.OptionsCount == 3 {
				// Дневной опрос → сразу заменяем
				if err := replaceParticipantDay(pollObj, oldUser, newUser); err != nil {
					sendQuietMessage(bot, chatID, fmt.Sprintf("Ошибка при замене: %v", err))
					return
				}
				// Просто сообщаем об успехе
				sendNormalMessage(bot, chatID, fmt.Sprintf("✅ @%s заменён на @%s (дневной опрос)", oldUser, newUser))

			} else {
				// Недельный → показываем клавиатуру
				sendReplaceDayKeyboard(bot, pollObj, oldUser, newUser, msg.ReplyToMessage.MessageID)
				// sendNormalMessage(bot, chatID, fmt.Sprintf("Выберите день для замены (@%s → @%s).", oldUser, newUser))
			}

		default:
			sendQuietMessage(bot, chatID, constants.MsgUnknownCommand)
		}

	} else {
		if oldMsgID, ok := lastQuietMessage[chatID]; ok {
			bot.Request(tgbotapi.NewDeleteMessage(chatID, oldMsgID))
			delete(lastQuietMessage, chatID)
		}
		text := strings.ToLower(strings.TrimSpace(msg.Text))
		text = sanitizeText(text)

		switch text {
		case "хуй":
			reply := "Сам ты хуй, а если нужна помощь, то:\n" + constants.MsgHelp
			sendQuietMessage(bot, chatID, reply)
		case "пидар", "пидор":
			reply := "Сам ты пидор, а если нужна помощь, то:\n" + constants.MsgHelp
			sendQuietMessage(bot, chatID, reply)
		default:
			// Не отвечаем
		}
	}
}

func deleteUserCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg.IsCommand() {
		del := tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID)
		bot.Request(del)
	}
}

func sendNormalMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	bot.Send(m)
}

func sendQuietMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	m.DisableNotification = true
	sent, err := bot.Send(m)
	if err != nil {
		log.Println("Ошибка отправки тихого сообщения:", err)
		return
	}
	lastQuietMessage[chatID] = sent.MessageID
}
