package main

import (
	"strings"

	"github.com/spf13/viper"
)

var config = viper.NewWithOptions(viper.EnvKeyReplacer(strings.NewReplacer(".", "_")))

func setConfig() {
	// Настройки переменных среды
	config.SetEnvPrefix("WB")
	config.AutomaticEnv()
}
