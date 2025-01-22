package main

import (
	"fmt"
	"log/slog"

	"github.com/e-vasilyev/wb-tool/internal/wbapi"
	"github.com/go-co-op/gocron"
)

// marketplaceStock описывает остаток по товару для известного sku
type marketplaceStock struct {
	amount uint32
}

// marketplaceStock описывает остатки по товарам на складах продавца
type marketplaceStocks struct {
	stocks map[string]*marketplaceStock
}

// getStock возвращает остаток по всем складам продавца для известного sku
func (m *marketplaceStocks) getStock(s string) uint32 {
	return m.stocks[s].amount
}

// count возвращает количество бракодов
func (m *marketplaceStocks) count() int {
	return len(m.stocks)
}

// newMarketplsceStocks создает спиоск остатков на складах продавца.
func newMarketplsceStocks(wbClient *wbapi.Client, skus []string) (*marketplaceStocks, error) {
	wbWarehouses, err := wbClient.GetWarehouses()
	if err != nil {
		return nil, err
	}

	result := &marketplaceStocks{stocks: make(map[string]*marketplaceStock)}

	for _, wbWarehouse := range wbWarehouses {
		wbStocks, err := wbClient.GetStocks(*wbWarehouse, skus)
		if err != nil {
			return nil, err
		}

		for _, wbStock := range wbStocks.Stocks {
			if _, ok := result.stocks[wbStock.Sku]; ok {
				result.stocks[wbStock.Sku].amount += wbStock.Amount
			} else {
				result.stocks[wbStock.Sku] = &marketplaceStock{amount: wbStock.Amount}
			}
		}
	}

	return result, nil
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

	stocks, err := newMarketplsceStocks(wbClient, skus)
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

// getSkusForDelete получает список skus для удаления
func (m *marketplaceStocks) getSkusForDelete() ([]string, error) {
	var res []string

	skus, err := pdb.getSkusMarketplaceStocksTable()
	if err != nil {
		return []string{}, err
	}
	slog.Info(fmt.Sprintf("Получено %d баркодов из БД остатков складов продавца", len(skus)))

	for _, sku := range skus {
		if _, ok := m.stocks[sku]; !ok {
			res = append(res, sku)
			slog.Info(fmt.Sprintf("Баркод %s пристутвует в БД остатков складов продавца, но отсутвует в магазине", sku))
		}
	}
	return res, nil
}
