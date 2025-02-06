package main

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

var config = viper.NewWithOptions(viper.EnvKeyReplacer(strings.NewReplacer(".", "_")))

func setConfig() {
	// Настройки переменных среды
	config.SetEnvPrefix("WB")
	config.AutomaticEnv()

	// Настройки базы данных
	config.SetDefault("database.name", "wb_tool")
	config.SetDefault("database.host", "localhost")
	config.SetDefault("database.port", "5432")
	config.SetDefault("database.username", "postgres")
	config.SetDefault("database.password", "postgres")
	config.Set("database.url", fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		config.GetString("database.username"),
		config.GetString("database.password"),
		config.GetString("database.host"),
		config.GetString("database.port"),
		config.GetString("database.name"),
	))

	// Настройки задач
	config.SetDefault("cron.content_cards_sync", "0 */4 * * *")
	config.SetDefault("cron.stoks_sync", "10 */2 * * *")

	// Настройка запросов
	config.SetDefault("statistics.date_from", "2023-11-01")
}
