package main

import (
	"fmt"
	"log/slog"

	"github.com/e-vasilyev/wb-tool/internal/wbapi"
)

// supplierStock описывает остаток по товару для известного sku
type supplierStock struct {
	quantity        uint32
	inWayToClient   uint32
	inWayFromClient uint32
	quantityFull    uint32
}

// supplierStocks описывает остатки по товарам на складах WB
type supplierStocks struct {
	stocks map[string]*supplierStock
}

// newSupplierStocks создает спиоск остатков на складах WB.
func newSupplierStocks(wbClient *wbapi.Client, dateFrom string) (*supplierStocks, error) {
	wbStocks, err := wbClient.GetStatisticsSupplierStock(dateFrom)
	if err != nil {
		return nil, err
	}

	result := &supplierStocks{stocks: make(map[string]*supplierStock)}

	for _, wbStock := range wbStocks {
		if _, ok := result.stocks[wbStock.Barcode]; ok {
			result.stocks[wbStock.Barcode].quantity += wbStock.Quantity
			result.stocks[wbStock.Barcode].quantityFull += wbStock.QuantityFull
			result.stocks[wbStock.Barcode].inWayFromClient += wbStock.InWayFromClient
			result.stocks[wbStock.Barcode].inWayToClient += wbStock.InWayToClient
		} else {
			result.stocks[wbStock.Barcode] = &supplierStock{
				quantity:        wbStock.Quantity,
				quantityFull:    wbStock.QuantityFull,
				inWayToClient:   wbStock.InWayToClient,
				inWayFromClient: wbStock.InWayFromClient,
			}
		}
	}

	return result, nil
}

// count возвращает количество бракодов
func (m *supplierStocks) count() int {
	return len(m.stocks)
}

// getSkusForDelete получает список skus для удаления
func (m *supplierStocks) getSkusForDelete() ([]string, error) {
	var res []string

	skus, err := pdb.getSkusSupplierStocksTable()
	if err != nil {
		return []string{}, err
	}
	slog.Info(fmt.Sprintf("Получено %d баркодов из БД остатков складов WB", len(skus)))

	for _, sku := range skus {
		if _, ok := m.stocks[sku]; !ok {
			res = append(res, sku)
			slog.Info(fmt.Sprintf("Баркод %s пристутвует в БД остатков складов WB, но отсутвует в магазине", sku))
		}
	}
	return res, nil
}
