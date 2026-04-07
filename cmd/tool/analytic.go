package main

import (
	"fmt"
	"log/slog"

	"github.com/e-vasilyev/wb-tool/internal/wbapi"
)

// wbStock описывает остаток по товару для известного размера
type wbStock struct {
	quantity        uint32
	inWayToClient   uint32
	inWayFromClient uint32
	quantityFull    uint32
}

// wbStocks описывает остатки по товарам на складах WB
type wbStocks struct {
	stocks map[uint32]*wbStock
}

// newReportWbStocks создает спиоск остатков на складах WB.
func newReportWbStocks(wbClient *wbapi.Client, nmIds []uint32, chrtIDs []uint32) (*wbStocks, error) {
	stocks, err := wbClient.GetStocksReport(nmIds, chrtIDs)
	if err != nil {
		return nil, err
	}

	result := &wbStocks{stocks: make(map[uint32]*wbStock)}

	for _, item := range stocks.Data.Items {
		if _, ok := result.stocks[item.ChrtID]; ok {
			result.stocks[item.ChrtID].quantity += item.Quantity
			result.stocks[item.ChrtID].quantityFull += item.Quantity + item.InWayFromClient + item.InWayToClient
			result.stocks[item.ChrtID].inWayFromClient += item.InWayFromClient
			result.stocks[item.ChrtID].inWayToClient += item.InWayToClient
		} else {
			result.stocks[item.ChrtID] = &wbStock{
				quantity:        item.Quantity,
				quantityFull:    item.Quantity + item.InWayFromClient + item.InWayToClient,
				inWayToClient:   item.InWayToClient,
				inWayFromClient: item.InWayFromClient,
			}
		}
	}

	return result, nil
}

// getChrtIDsForDelete получает список chrtID для удаления
func (m *wbStocks) getChrtIDsForDelete() ([]uint32, error) {
	var res []uint32

	chrtIDs, err := pdb.getChrtIDsWbStocksTable()
	if err != nil {
		return []uint32{}, err
	}
	slog.Info(fmt.Sprintf("Получено %d размеров из БД остатков складов WB", len(chrtIDs)))

	for _, chrtID := range chrtIDs {
		if _, ok := m.stocks[chrtID]; !ok {
			res = append(res, chrtID)
			slog.Info(fmt.Sprintf("ID размера %d пристутвует в БД остатков складов WB, но отсутвует в магазине", chrtID))
		}
	}
	return res, nil
}
