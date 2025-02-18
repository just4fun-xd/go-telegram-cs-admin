package bot

import (
	"log"
	"regexp"
	"strings"

	"go-telegram-cs-admin/internal/constants"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var lastQuietMessage = make(map[int64]int)

// sanitizeText удаляет знаки препинания и лишние символы
func sanitizeText(text string) string {
	re := regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	return re.ReplaceAllString(text, "")
}

func HandleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	defer deleteUserCommand(bot, msg)

	if msg.IsCommand() {
		// Удаляем предыдущее "тихое" сообщение (help/unknown)
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
		default:
			sendQuietMessage(bot, chatID, constants.MsgUnknownCommand)
		}
	} else {
		// Обработка текстовых сообщений (не команд)
		if oldMsgID, ok := lastQuietMessage[chatID]; ok {
			bot.Request(tgbotapi.NewDeleteMessage(chatID, oldMsgID))
			delete(lastQuietMessage, chatID)
		}
		// Приводим текст к нижнему регистру и убираем пробелы
		text := strings.ToLower(strings.TrimSpace(msg.Text))
		// Также очищаем от лишней пунктуации
		text = sanitizeText(text)

		switch text {
		case "хуй":
			reply := "Сам ты хуй, а если нужна помощь, то:\n" + constants.MsgHelp
			sendQuietMessage(bot, chatID, reply)
		case "пидар", "пидор":
			reply := "Сам ты пидор, а если нужна помощь, то:\n" + constants.MsgHelp
			sendQuietMessage(bot, chatID, reply)
		default:
			// Если текст не соответствует ни одному из заданных вариантов, не отвечаем.
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
