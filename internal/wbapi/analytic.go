package wbapi

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	analyticsPathPing                 string = "ping"
	analyticsPathStocksReport         string = "api/analytics/v1/stocks-report/wb-warehouses"
	analyticsStocksReportRequestLimit uint32 = 250000
)

// analyticsRequestTicker канал контролирующй количество отправленных запросов в минуту.
// 1 запрос в минуту в разделе контента
var analyticsRequestTicker <-chan time.Time = time.NewTicker(time.Minute).C

// stocksReportDataItem описывает блок items остатков на складах WB
type stocksReportDataItem struct {
	NmID            uint32 `json:"nmId"`
	ChrtID          uint32 `json:"chrtId"`
	WarehouseID     uint32 `json:"warehouseId"`
	WarehouseName   string `json:"warehouseName"`
	RegionName      string `json:"regionName"`
	Quantity        uint32 `json:"quantity"`
	InWayToClient   uint32 `json:"inWayToClient"`
	InWayFromClient uint32 `json:"inWayFromClient"`
}

// StocksReportData описывает блок data остатков на складах WB
type StocksReportData struct {
	Items []stocksReportDataItem `json:"items"`
}

// StocksReport описывает остатки на складах WB
type StocksReport struct {
	Data StocksReportData `json:"data"`
}

// stocksReportRequest описывает блок cursor в запросе
type stocksReportRequest struct {
	NmIds   []uint32 `json:"nmIds"`
	ChrtIDs []uint32 `json:"chrtIds,omitempty"`
	Limit   uint32   `json:"limit,omitempty"`
	Offset  uint32   `json:"offset,omitempty"`
}

// GetStocksReport возвращает список остатков со складов WB
// Из-за ограничения количества позиций в ответе парметром analyticsStocksReportRequestLimit
// запрос выполняется с пагинацией
func (c *Client) GetStocksReport(nmIds []uint32, chrtIDs []uint32) (*StocksReport, error) {
	c.logger.Debug("Получение данных остатков по складам WB")

	stocksReport := &StocksReport{
		Data: StocksReportData{
			Items: []stocksReportDataItem{},
		},
	}
	url := fmt.Sprintf("%s/%s", c.baseURL.analytics, analyticsPathStocksReport)

	var offset uint32 = 0
	body := &stocksReportRequest{
		NmIds:   nmIds,
		ChrtIDs: chrtIDs,
		Limit:   analyticsStocksReportRequestLimit,
		Offset:  offset,
	}

	for {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		page, err := c.getStocksReport(url, jsonBody)

		if err != nil {
			return nil, err
		}

		stocksReport.Data.Items = append(stocksReport.Data.Items, page.Data.Items...)
		if uint32(len(page.Data.Items)) <= analyticsStocksReportRequestLimit {
			break
		}

		offset += analyticsStocksReportRequestLimit
		body.Offset = offset
	}

	return stocksReport, nil
}

// getStocksReport возвращает список остатков со складов WB
// Запрос выдаст ограниченное количество элементов
// в зависимости от jsonBody и параметра analyticsStocksReportRequestLimit
func (c *Client) getStocksReport(url string, jsonBody []byte) (*StocksReport, error) {
	stocksReport := &StocksReport{
		Data: StocksReportData{
			Items: []stocksReportDataItem{},
		},
	}

	res, err := c.postRequest(url, jsonBody, analyticsRequestTicker)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var code int
	if code, err = respCodeCheck(res); err != nil {
		return nil, err
	}

	if code != 204 {
		err = json.NewDecoder(res.Body).Decode(stocksReport)
	}

	return stocksReport, err
}
