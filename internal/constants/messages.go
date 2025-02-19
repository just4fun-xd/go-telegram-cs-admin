package constants

const (
	MsgStart = "👋 Привет! Я бот для организации встреч. Используй /poll или /poll_day, чтобы создать опрос."
	MsgHelp  = `📜 Список команд:
/poll - Создать еженедельный опрос (5 вариантов)
/poll_day DD.MM - Создать дневной опрос (3 варианта: да, нет, посмотреть)
/help - Вывести справку`
	MsgUnknownCommand = "⚠️ Я не понимаю. Воспользуйтесь командой /help."

	MsgReplaceFormat = "Формат: /replace @OldUser @NewUser"
	MsgReplyClose    = "Команда /close_pool должна быть ответом на сообщение с опросом."
	MsgPollClosed    = "✅ Голосование закрыто."
	MsgPollCreated   = "✅ Опрос создан. Выбирайте удобные дни!" // для /poll
	MsgMeetingPlace  = "📍 Встреча пройдёт в компьютерном клубе Molegan arena по адресу Каинская, 3 https://2gis.ru/novosibirsk/firm/70000001052475116"
)
