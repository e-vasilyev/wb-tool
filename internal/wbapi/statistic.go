package wbapi

import (
	"time"
)

const (
	statisticsPathPing string = "ping"
)

// statisticsRequestTicker канал содержащий количество отправленных запросов
// 1 запрос в минуту в разделе статистика.
var statisticsRequestTicker <-chan time.Time = time.NewTicker(time.Millisecond * 500).C
