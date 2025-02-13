package bot

import (
	"log"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"go-telegram-cs-admin/internal/constants"
)

var lastQuietMessage = make(map[int64]int)

func sanitizeText(text string) string {
	// Убираем знаки препинания и символы (оставляем буквы и цифры)
	re := regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	return re.ReplaceAllString(text, "")
}

func HandleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	defer deleteUserCommand(bot, msg)

	if msg.IsCommand() {
		// Если команда, удаляем предыдущее "тихое" сообщение и обрабатываем команду.
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
		// Если это не команда, проверяем, содержит ли сообщение упоминание бота.
		text := strings.ToLower(strings.TrimSpace(msg.Text))
		if msg.Entities != nil {
			for _, entity := range msg.Entities {
				if entity.Type == "mention" {
					// Извлекаем упоминание
					mention := text[entity.Offset : entity.Offset+entity.Length]
					// Если упоминание соответствует нашему боту, удаляем его из текста
					if mention == "@"+strings.ToLower(bot.Self.UserName) {
						text = strings.ReplaceAll(text, mention, "")
					}
				}
			}
		}
		// Обрезаем лишние пробелы после удаления упоминания.
		text = strings.TrimSpace(text)

		// Обрабатываем очищенный текст
		switch text {
		case "хуй":
			reply := "Сам ты хуй, а если нужна помощь, то:\n" + constants.MsgHelp
			sendQuietMessage(bot, chatID, reply)
		case "пидар", "пидор":
			reply := "Сам ты пидор, а если нужна помощь, то:\n" + constants.MsgHelp
			sendQuietMessage(bot, chatID, reply)
		default:
			sendQuietMessage(bot, chatID, constants.MsgUnknownCommand)
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
