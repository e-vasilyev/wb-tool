package main

import (
	"fmt"
	"log/slog"

	"github.com/e-vasilyev/wb-tool/internal/wbapi"
	"github.com/go-co-op/gocron"
)

// marketplaceStock описывает остаток по товару для известного sku
type marketplaceStock struct {
	sku    string
	amount uint32
}

// stocksSync синхронизирует остатки по карточкам
func stocksSync(wbClient *wbapi.Client, job gocron.Job) {
	skusRows, err := pdb.getContentSkusTable()
	if err != nil {
		slog.Error(fmt.Sprintf("При получении списка баркодов из БД произошла ошибка %s", err.Error()))
		return
	}

	var skus []string

	for _, row := range skusRows {
		skus = append(skus, row.Sku)
	}

	stocks, err := getMarketplaceStocks(wbClient, skus)
	if err != nil {
		slog.Error(fmt.Sprintf("При получении остатков произошла ошибка %s", err.Error()))
		return
	}

	if err := pdb.syncMarketplaceStocks(stocks, skusRows); err != nil {
		slog.Error(fmt.Sprintf("При синхронизации остатков складов продавца произошла ошибка %s", err.Error()))
		return
	}

	slog.Info(fmt.Sprintf("Следующий запуск задачи %s в %s", job.GetName(), job.NextRun()))
}

// getStocks получает список остатков через API WB
func getMarketplaceStocks(wbClient *wbapi.Client, skus []string) (map[string]uint32, error) {
	wbWarehouses, err := wbClient.GetWarehouses()
	if err != nil {
		return nil, err
	}

	var skusMap map[string]uint32 = make(map[string]uint32)

	for _, wbWarehouse := range wbWarehouses {
		wbStocks, err := wbClient.GetStocks(*wbWarehouse, skus)
		if err != nil {
			return nil, err
		}

		for _, wbStock := range wbStocks.Stocks {
			if _, ok := skusMap[wbStock.Sku]; ok {
				skusMap[wbStock.Sku] += wbStock.Amount
			} else {
				skusMap[wbStock.Sku] = wbStock.Amount
			}
		}
	}

	return skusMap, nil
}
