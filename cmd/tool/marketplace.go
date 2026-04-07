package main

import (
	"fmt"
	"log/slog"

	"github.com/e-vasilyev/wb-tool/internal/wbapi"
	"github.com/go-co-op/gocron"
)

// marketplaceStock описывает остаток по товару для известного chrtId (id размера)
type marketplaceStock struct {
	amount uint32
}

// marketplaceStocks описывает остатки по товарам на складах продавца
type marketplaceStocks struct {
	stocks map[uint32]*marketplaceStock
}

// getStock возвращает остаток по всем складам продавца для известного chrtId (id размера)
func (m *marketplaceStocks) getStock(s uint32) uint32 {
	return m.stocks[s].amount
}

// count возвращает количество chrtId
func (m *marketplaceStocks) count() int {
	return len(m.stocks)
}

// newMarketplsceStocks создает спиоск остатков на складах продавца.
func newMarketplsceStocks(wbClient *wbapi.Client, chrtIDs []uint32) (*marketplaceStocks, error) {
	wbWarehouses, err := wbClient.GetWarehouses()
	if err != nil {
		return nil, err
	}

	result := &marketplaceStocks{stocks: make(map[uint32]*marketplaceStock)}

	for _, wbWarehouse := range wbWarehouses {
		wbStocks, err := wbClient.GetStocks(*wbWarehouse, chrtIDs)
		if err != nil {
			return nil, err
		}

		for _, wbStock := range wbStocks.Stocks {
			if _, ok := result.stocks[wbStock.ChrtID]; ok {
				result.stocks[wbStock.ChrtID].amount += wbStock.Amount
			} else {
				result.stocks[wbStock.ChrtID] = &marketplaceStock{amount: wbStock.Amount}
			}
		}
	}

	return result, nil
}

// stocksSync синхронизирует остатки по карточкам
func stocksSync(wbClient *wbapi.Client, job gocron.Job) {
	defer slog.Info(fmt.Sprintf("Следующий запуск задачи '%s' в %s", job.GetName(), job.NextRun()))

	sizesRows, err := pdb.getContentSizesTable()
	if err != nil {
		slog.Error(fmt.Sprintf("При получении списка размеров из БД произошла ошибка %s", err.Error()))
		return
	}

	var chrtIDs []uint32

	for _, row := range sizesRows {
		chrtIDs = append(chrtIDs, row.ChrtID)
	}

	marketplsceStocks, err := newMarketplsceStocks(wbClient, chrtIDs)
	if err != nil {
		slog.Error(fmt.Sprintf("При получении остатков склада продавца произошла ошибка %s", err.Error()))
		return
	}
	slog.Info(fmt.Sprintf("Получено %d размеров", marketplsceStocks.count()))

	if err := pdb.syncMarketplaceStocks(marketplsceStocks, sizesRows); err != nil {
		slog.Error(fmt.Sprintf("При синхронизации остатков складов продавца произошла ошибка %s", err.Error()))
		return
	}

	var nmIDs []uint32

	for _, row := range sizesRows {
		nmIDs = append(nmIDs, row.NmID)
	}

	wbStocks, err := newReportWbStocks(wbClient, nmIDs, chrtIDs)
	if err != nil {
		slog.Error(fmt.Sprintf("При получении остатков складов WB произошла ошибка %s", err.Error()))
		return
	}

	if err := pdb.syncWbStocks(wbStocks, sizesRows); err != nil {
		slog.Error(fmt.Sprintf("При синхронизации остатков складов WB произошла ошибка %s", err.Error()))
		return
	}
}

// getChrtIDsForDelete получает список chrtId для удаления
func (m *marketplaceStocks) getChrtIDsForDelete() ([]uint32, error) {
	var res []uint32

	chrtIDs, err := pdb.getChrtIDsMarketplaceStocksTable()
	if err != nil {
		return []uint32{}, err
	}
	slog.Info(fmt.Sprintf("Получено %d размеров из БД остатков складов продавца", len(chrtIDs)))

	for _, chrtID := range chrtIDs {
		if _, ok := m.stocks[chrtID]; !ok {
			res = append(res, chrtID)
			slog.Info(fmt.Sprintf("ID размера %d пристутвует в БД остатков складов продавца, но отсутвует в магазине", chrtID))
		}
	}
	return res, nil
}
