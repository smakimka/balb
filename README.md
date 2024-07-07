# BALB - Birthday ALert Bot

## Запуск

 1. Создать бота в телеграмм
 2. Указать в файле .bot_env токен авторизации (токен, который будет бот спрашивать у всех во время регистрации (чтобы все не пользовались)), токен бота, полученный от BotFather и chat id администратора (т.к. боты в тг не могут создавать группы (я почти уверен), то бот будет просить администратора это сделать и потом отметить группу командой) 
 3. Выполнить docker compose build
 4. Выполнить dokcer compose up
 5. Должно работать
 
 ## Запуск тестов
 ```bash
 go test ./internal/server/handlers 
```

## Схема каталогов
cmd - исполняемые файлы (main.go), которые собираются из всего остального.  Bot  и Server  соответственно
internal - Основная директория файлов проекта, делится на
 - bot - Основная директория сервиса bot
	 - bot - пакет телеграм бота
	 - dialog - пакет отвечающий за сбор регистрацию и авторизацию пользователей
	 - handlers - пакет с хендлерами сервера
	 - notifier - пакет с модулем, выполняющим фоновые задачи (попросить создать директорию, отправить приглашения)
	 - router - пакет с роутером сервиса
	 - storage - пакет с хранилищем данных сервиса
 - server - основная директория сервиса server
	 - handlers - пакет с хендлерами сервера
	 - notifier - пакет с модулем, выполняющим фоновые задачи (отправка запроса на уведомление)
	 - router - пакет с роутером сервиса
	 - storage - пакет с хранилищем данных сервиса
 - model - директория пакета model, содержащего в себе основные структуры данных общие для обоих сервисов
## Схема работы
Сервис разделен на 2 маленьких и базу данных
server - основной сервис, в него все "фронты" должны отправлять данные о пользователях и получать от него запрос на уведомление о др. Ответственность за доставку несут они.
bot - тг фронт сервиса, осуществляет регистрацию и отправку уведомлений, как - его дело.
