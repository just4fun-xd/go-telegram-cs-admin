package constants

const (
	MsgStart          = "👋 Привет! Я бот для организации встреч. Используй /poll, чтобы создать опрос."
	MsgUnknownCommand = "⚠️ Я не понимаю. Воспользуйтесь командой /help."
	MsgReplaceFormat  = "Формат: /replace @OldUser @NewUser"
	MsgReplyClose     = "Команда /close_pool должна быть ответом на сообщение с опросом."
	MsgPollCreated    = "✅ Опрос создан. Выбирайте удобные дни!"
	MsgPollClosed     = "✅ Голосование закрыто."
	MsgPlayersReply   = "Команда /players должна быть ответом на сообщение с опросом."
	MsgHelp           = `📜 Список команд:
/poll - создать опрос
/help - помощь
/replace @OldUser @NewUser - заменить участника
/close_pool - закрыть голосование (нужно ответить на сообщение с опросом)
/players - вывести список всех участников (нужно ответить на сообщение с опросом)`
)
