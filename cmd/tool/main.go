package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/e-vasilyev/wb-tool/internal/wbapi"
	"github.com/go-co-op/gocron"
)

var (
	logLevelString string   = os.Getenv("WB_LOG_LEVEL")
	pdb            *pClinet = &pClinet{pool: nil, ctx: context.Background()}
)

func main() {
	// Инициализация логирования
	logLevel := slog.LevelInfo
	switch strings.ToLower(logLevelString) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// Настройка конфигурации приложения
	setConfig()

	// Получение токена
	token := config.GetString("token")

	// Создание клиента WB API
	wbClient := wbapi.NewClientWithOptions(token, wbapi.SetClientLogger(logger))
	if err := wbClient.Ping(); err != nil {
		slog.Error(fmt.Sprintf("При подключении к API получена критическая ошибка %s", err.Error()))
		os.Exit(1)
	} else {
		slog.Info("Проверка подключения к API прошла успешно")
	}

	// Подключение к БД
	pool, err := connectToDB(pdb.ctx)
	if err != nil {
		slog.Error(fmt.Sprintf("При подключении к базе данных получена критическая ошибка %s", err.Error()))
		os.Exit(1)
	}
	pdb.pool = pool
	defer pdb.pool.Close()

	// Миграция базы данных
	if err := pdb.migration(); err != nil {
		slog.Error(fmt.Sprintf("При миграции данных получена критическая ошибка: %s", err.Error()))
		os.Exit(1)
	}

	// Запуск задач
	scheduler := gocron.NewScheduler(time.Local)

	jobContentSync, err := scheduler.Cron(config.GetString("cron.content_cards_sync")).DoWithJobDetails(contentSync, wbClient)
	jobContentSync.Name("Синхронизация карточек")
	jobContentSync.SingletonMode()

	jobStoksSync, err := scheduler.Cron(config.GetString("cron.stoks_sync")).DoWithJobDetails(stocksSync, wbClient)
	jobStoksSync.Name("Синхронизация остатков")
	jobStoksSync.SingletonMode()

	scheduler.RegisterEventListeners(
		gocron.BeforeJobRuns(func(jobName string) {
			slog.Info(fmt.Sprintf("Запуск задачи: %s", jobName))
		}))
	scheduler.StartBlocking()
}
