package wbapi

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	marketplacePathPing       string = "ping"
	marketplacePathWarehouses string = "api/v3/warehouses"
	marketplacePathStocks     string = "api/v3/stocks"
	marketplaceChrtIDsLimit   int    = 1000
)

// marketplaceRequestTicker канал контролирующй количество отправленных запросов в минуту.
// 120 запросов в минуту в разделе маркетплейса (2 раза в секунду).
var marketplaceRequestTicker <-chan time.Time = time.NewTicker(time.Millisecond * 500).C

// Warehouse описывает склад продавца
type Warehouse struct {
	Name         string `json:"name"`
	OfficeID     uint32 `json:"officeId"`
	ID           uint32 `json:"id"`
	CargoType    uint32 `json:"cargoType"`
	DeliveryType uint32 `json:"deliveryType"`
}

// Stock описывает остаток на складе продавца
type Stock struct {
	ChrtID uint32 `json:"chrtId"`
	Amount uint32 `json:"amount"`
}

// Stocks описывает остатки на складе продваца
type Stocks struct {
	Stocks []Stock `json:"stocks"`
}

// stockRequest описывает тело запроса для получению остатков
type stockRequest struct {
	ChrtID []uint32 `json:"chrtIds"`
}

// GetWarehouses получает список складов продавца
func (c *Client) GetWarehouses() ([]*Warehouse, error) {
	c.logger.Debug("Получение списка складов продавца")

	var warehouses []*Warehouse

	url := fmt.Sprintf("%s/%s", c.baseURL.marketplace, marketplacePathWarehouses)

	res, err := c.getRequest(url, marketplaceRequestTicker)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if _, err := respCodeCheck(res); err != nil {
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(&warehouses)
	if err != nil {
		return nil, err
	}

	return warehouses, nil
}

// GetStocks получает остатки по слкаду продавца,
// можно передать массив больше marketplaceChrtIDsLimit, в этом случае запросы разделятся на части
func (c *Client) GetStocks(warehouse Warehouse, chrtIDs []uint32) (*Stocks, error) {
	c.logger.Debug(fmt.Sprintf("Получение остатка на складе продавца: %s ", warehouse.Name))

	var stocks *Stocks = &Stocks{}

	if len(chrtIDs) == 0 {
		c.logger.Warn("Пустой список id размеров для получения остатков")
		return &Stocks{Stocks: []Stock{}}, nil
	}

	url := fmt.Sprintf("%s/%s/%d", c.baseURL.marketplace, marketplacePathStocks, warehouse.ID)

	countPages := len(chrtIDs)/marketplaceChrtIDsLimit + 1
	var page [][]uint32 = make([][]uint32, countPages)

	for i := 0; i < countPages; i++ {
		start := i * marketplaceChrtIDsLimit
		stop := (i + 1) * marketplaceChrtIDsLimit
		if stop > len(chrtIDs) {
			stop = len(chrtIDs)
		}

		page[i] = chrtIDs[start:stop]

		body := &stockRequest{
			ChrtID: page[i],
		}

		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		stocksPage, err := c.getStocks(url, jsonBody)
		if err != nil {
			return nil, err
		}

		stocks.Stocks = append(stocks.Stocks, stocksPage.Stocks...)
	}

	return stocks, nil
}

// getStocks  получает остатки по слкаду продавца, длина массива в теле запроса ограничена
func (c *Client) getStocks(url string, jsonBody []byte) (*Stocks, error) {
	var stocks *Stocks

	res, err := c.postRequest(url, jsonBody, marketplaceRequestTicker)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if _, err := respCodeCheck(res); err != nil {
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(&stocks)

	return stocks, err
}
