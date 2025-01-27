package wbapi

import (
	"encoding/json"
	"fmt"
)

const (
	marketplacePathPing          string = "ping"
	marketplacePathWarehouses    string = "api/v3/warehouses"
	marketplacePathStocks        string = "api/v3/stocks"
	marketplaceSkusLimit         int    = 1000
	marketplaceRequestsPerMinute uint8  = 200 // Количество разрешенных запросов в минуту
)

// marketplaceRequestCount канал содержащий количество отправленных запросов
// в разделе маркетплейса.
var marketplaceRequestCount chan uint8 = make(chan uint8, marketplaceRequestsPerMinute)

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
	Sku    string `json:"sku"`
	Amount uint32 `json:"amount"`
}

// Stocks описывает остатки на складе продваца
type Stocks struct {
	Stocks []Stock `json:"stocks"`
}

// stockRequest описывает тело запроса для получению остатков
type stockRequest struct {
	Skus []string `json:"skus"`
}

// GetWarehouses получает список складов продавца
func (c *Client) GetWarehouses() ([]*Warehouse, error) {
	c.logger.Debug("Получение списка складов продавца")

	var warehouses []*Warehouse

	url := fmt.Sprintf("%s/%s", c.baseURL.marketplace, marketplacePathWarehouses)

	res, err := c.getRequest(url, marketplaceRequestCount)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := respCodeCheck(res); err != nil {
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(&warehouses)
	if err != nil {
		return nil, err
	}

	return warehouses, nil
}

// GetStocks получает остатки по слкаду продавца,
// можно передать массив больше 1000, в этом случае запросы разделятся на части
func (c *Client) GetStocks(warehouse Warehouse, skus []string) (*Stocks, error) {
	c.logger.Debug(fmt.Sprintf("Получение остатка на складе продавца: %s ", warehouse.Name))

	var stocks *Stocks = &Stocks{}

	url := fmt.Sprintf("%s/%s/%d", c.baseURL.marketplace, marketplacePathStocks, warehouse.ID)

	countPages := len(skus)/marketplaceSkusLimit + 1
	var pSkus [][]string = make([][]string, countPages)

	for i := 0; i < countPages; i++ {
		start := i * marketplaceSkusLimit
		stop := (i + 1) * marketplaceSkusLimit
		if stop > len(skus) {
			stop = len(skus)
		}

		pSkus[i] = skus[start:stop]

		body := &stockRequest{
			Skus: pSkus[i],
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

	res, err := c.postRequest(url, jsonBody, marketplaceRequestCount)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if err := respCodeCheck(res); err != nil {
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(&stocks)

	return stocks, err
}
