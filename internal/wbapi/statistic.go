package wbapi

import (
	"encoding/json"
	"fmt"
)

const (
	statisticsPathPing           string = "ping"
	statisticsPathSupplierStocks string = "api/v1/supplier/stocks"
	statisticsRequestsPerMinute  uint8  = 1 // Количество разрешенных запросов в минуту
)

// statisticsRequestCount канал содержащий количество отправленных запросов
// в разделе статистика.
var statisticsRequestCount chan uint8 = make(chan uint8, statisticsRequestsPerMinute)

// StatisticsSupplierStock описывает остатки на складе
type StatisticsSupplierStock struct {
	LastChangeDate  string  `json:"lastChangeDate"`
	WarehouseName   string  `json:"warehouseName"`
	SupplierArticle string  `json:"supplierArticle"`
	NmID            uint32  `json:"nmId"`
	Barcode         string  `json:"barcode"`
	Quantity        uint32  `json:"quantity"`
	InWayToClient   uint32  `json:"inWayToClient"`
	InWayFromClient uint32  `json:"inWayFromClient"`
	QuantityFull    uint32  `json:"quantityFull"`
	Category        string  `json:"category"`
	Subject         string  `json:"subject"`
	Brand           string  `json:"brand"`
	TechSize        string  `json:"techSize"`
	Price           float32 `json:"Price"`
	Discount        uint32  `json:"Discount"`
	IsSupply        bool    `json:"isSupply"`
	IsRealization   bool    `json:"isRealization"`
	SCCode          string  `json:"SCCode"`
}

// GetStatisticsSupplierStock возвращает список остатков со складов WB
func (c *Client) GetStatisticsSupplierStock(dateFrom string) ([]*StatisticsSupplierStock, error) {
	c.logger.Debug("Получение данных остатков по складам WB")

	var statisticsSupplierStocks []*StatisticsSupplierStock

	url := fmt.Sprintf("%s/%s?dateFrom=%s", c.baseURL.statistics, statisticsPathSupplierStocks, dateFrom)

	res, err := c.getRequest(url, statisticsRequestCount)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := respCodeCheck(res); err != nil {
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(&statisticsSupplierStocks)
	if err != nil {
		return nil, err
	}

	return statisticsSupplierStocks, nil
}
