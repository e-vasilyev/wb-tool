# WB-TOOL

Приложение wb-tool предназначена для автоматизации некоторых действий в кабинете WB, а так же ведения статистики. Полученные данные записываются в базу данных postgresql.

## Возможности

Информация по карточкам находящимся в продаже и в корзние
Информация по остаткам

## Сборка приложения

```bash
go build -v -o wb-tool ./cmd/tool
```

## Настройка

Настройка приложения осуществляется при помощи переменных среды.

| Переменная                 | Значение по умолчанию | Описание                                                                       |
| -------------------------- | --------------------- | ------------------------------------------------------------------------------ |
| WB_CRON_CONTENT_CARDS_SYNC | 0 */4 * * *           | Расписание запуска синхронизации карточек                                      |
| WB_CRON_STOKS_SYNC         | 0 */2 * * *           | Расписание запуска синхронизации остатков                                      |
| WB_DATABASE_NAME           | wb_tool               | Имя базы данных                                                                |
| WB_DATABASE_HOST           | localhost             | Хост базы данных                                                               |
| WB_DATABASE_PORT           | 5432                  | Порт базы данных                                                               |
| WB_DATABASE_USERNAME       | postgres              | Пользователь базы данных                                                       |
| WB_DATABASE_PASSWORD       | postgres              | Пароль пользователя базы данных                                                |
| WB_LOG_LEVEL               | Info                  | Уровень логирования. Доступные уровни: Info, Warn, Error, Debug                |
| WB_STATISTICS_DATE_FROM    | 2023-11-01            | Дата с которой получать отстатки по карточка. Желтально указать наиболее ранюю |
| WB_TOKEN                   |                       | Токен доступа к API WB с правами Контент, Маркетплейс, Статистика              |
