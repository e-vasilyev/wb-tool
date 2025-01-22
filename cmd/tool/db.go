package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/e-vasilyev/wb-tool/assets"
	"github.com/pressly/goose/v3"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// pClinet описывает подключение к базе данных postgresql
type pClinet struct {
	pool *pgxpool.Pool
	ctx  context.Context
}

// contentSkusTable описывает структуру таблицы wb_content_skus
type contentSkusTable struct {
	NmID uint32 `db:"nm_id"`
	Sku  string `db:"sku"`
}

// connectToDB открывает пул соединений
func connectToDB(ctx context.Context) (*pgxpool.Pool, error) {
	var url = config.GetString("database.url")
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}

// migrationDB запускает миграции баз данных
func (p *pClinet) migration() error {
	var migrations = &assets.Migrations

	slog.Info("Миграция базы данных")

	goose.SetBaseFS(migrations)
	goose.SetTableName("wb_tool_goose_db_version")
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	db := stdlib.OpenDBFromPool(p.pool)
	if err := goose.Up(db, "migrations"); err != nil {
		return err
	}

	return nil
}

// syncContentCards синхронизирует данные полученные с api в БД
func (p *pClinet) syncContentCards(cs *contentCards) error {
	tx, err := p.pool.Begin(p.ctx)
	if err != nil {
		slog.Error(fmt.Sprintf("При создании транзакции произошла ошибка %s", err.Error()))
		return err
	}

	defer tx.Rollback(p.ctx)

	for _, card := range cs.cards {
		if cs.trashed {
			err = p.upsertContentTrashedCard(tx, *card)
		} else {
			err = p.upsertContentCard(tx, *card)
		}

		if err != nil {
			return err
		}

		if err := p.upsetSkus(tx, *card); err != nil {
			return err
		}
	}

	ids, err := cs.getNmIDsForDelete()

	if err != nil {
		slog.Error(fmt.Sprintf("При получении nmID для удаления произошла ошибка %s", err.Error()))
		return err
	}

	if err := p.markAsDeleted(tx, ids); err != nil {
		return err
	}

	if err := tx.Commit(pdb.ctx); err != nil {
		slog.Error(fmt.Sprintf("При коммите изменений в БД произошла ошибка %s", err.Error()))
		return err
	}
	slog.Info(fmt.Sprintf("Карточки успешно синхронизировны"))

	return nil
}

// upsertContentTrashedCard добавляет запись карторчки из корзины в БД
func (p *pClinet) upsertContentTrashedCard(tx pgx.Tx, card contentCard) error {
	_, err := tx.Exec(
		p.ctx,
		`INSERT INTO wb_content_cards (nm_id, vendor_code, subject_id, subject_name, trashed_at, trashed, updated_timestamp) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (nm_id) DO UPDATE
				SET vendor_code = $2, subject_id = $3, subject_name = $4, 
					trashed_at = $5, trashed = $6, updated_timestamp = $7`,
		card.nmID, card.vendorCode, card.subjectID,
		card.subjectName, card.trashedAt, true,
		time.Now().UTC().Format("2006-01-02 03:04:05"),
	)
	if err != nil {
		slog.Error(fmt.Sprintf("При записи карточки %d в базу данных возникла ошибка %s", card.nmID, err.Error()))
		return err
	}

	return nil
}

// upsertContentCard добавляет запись карторчки в БД
func (p *pClinet) upsertContentCard(tx pgx.Tx, card contentCard) error {
	_, err := tx.Exec(
		p.ctx,
		`INSERT INTO wb_content_cards (nm_id, imt_id, vendor_code, subject_id, subject_name, brand, title, trashed, updated_timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (nm_id) DO UPDATE
				SET imt_id = $2, vendor_code = $3, subject_id = $4, subject_name = $5, 
					brand = $6, title = $7, trashed = $8, updated_timestamp = $9`,
		card.nmID, card.imtID, card.vendorCode, card.subjectID,
		card.subjectName, card.brand, card.title, false,
		time.Now().UTC().Format("2006-01-02 03:04:05"),
	)
	if err != nil {
		slog.Error(fmt.Sprintf("При записи карточки %d в базу данных возникла ошибка %s", card.nmID, err.Error()))
		return err
	}

	return nil
}

// upsetSkus обновляет записи по бракодам в БД
func (p *pClinet) upsetSkus(tx pgx.Tx, card contentCard) error {
	for _, sku := range card.skus {
		_, err := tx.Exec(
			p.ctx,
			`INSERT INTO wb_content_skus (sku, nm_id)
				VALUES ($1, $2)
				ON CONFLICT (sku) DO NOTHING`,
			sku, card.nmID,
		)
		if err != nil {
			slog.Error(fmt.Sprintf("При записи баркода %s в базу дунных возникла ошибка %s", sku, err.Error()))
			return err
		}
	}

	return nil
}

// deleteSku удаляет записи по бракодам в БД
func (p *pClinet) deleteSku(tx pgx.Tx, sku string) error {
	_, err := tx.Exec(
		p.ctx,
		`DELETE FROM wb_content_skus WHERE sku = $1`,
		sku,
	)
	if err != nil {
		slog.Error(fmt.Sprintf("При удалении баркода %s возникла ошибка %s", sku, err.Error()))
		return err
	}

	return nil
}

// deleteSku удаляет записи по остаткам на складах продавца в БД
func (p *pClinet) deleteMarketplaceStockBySku(tx pgx.Tx, sku string) error {
	_, err := tx.Exec(
		p.ctx,
		`DELETE FROM wb_marketplace_stocks WHERE sku = $1`,
		sku,
	)
	if err != nil {
		slog.Error(fmt.Sprintf("При удалении остатка по баркоду %s возникла ошибка %s", sku, err.Error()))
		return err
	}

	return nil
}

// getContentSkusTable возвращает содержимое таблицы wb_content_skus из БД
func (p *pClinet) getContentSkusTable() ([]*contentSkusTable, error) {
	rows, err := p.pool.Query(
		p.ctx, "SELECT sku, nm_id FROM wb_content_skus NATURAL JOIN wb_content_cards WHERE deleted is false",
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	skus, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[contentSkusTable])
	if err != nil {
		return nil, err
	}

	return skus, nil
}

// getTrashedNmIDsConentCardsTable получает массив nmID из таблицы карточек в корзине
func (p *pClinet) getTrashedNmIDsConentCardsTable() ([]uint32, error) {
	rows, err := p.pool.Query(
		p.ctx, "SELECT nm_id FROM wb_content_cards WHERE trashed is true and deleted is false",
	)
	if err != nil {
		return []uint32{}, err
	}

	defer rows.Close()

	nmIDs, err := pgx.CollectRows(rows, pgx.RowTo[uint32])
	if err != nil {
		return []uint32{}, err
	}

	return nmIDs, nil
}

// getNmIDsConentCardsTable получает массив nmID из таблицы карточек
func (p *pClinet) getNmIDsConentCardsTable() ([]uint32, error) {
	rows, err := p.pool.Query(
		p.ctx, "SELECT nm_id FROM wb_content_cards WHERE trashed is false and deleted is false",
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	nmIDs, err := pgx.CollectRows(rows, pgx.RowTo[uint32])
	if err != nil {
		return nil, err
	}

	return nmIDs, nil
}

// Помечает как удаленные карточки с указанными nmID
func (p *pClinet) markAsDeleted(tx pgx.Tx, ids []uint32) error {
	for _, id := range ids {
		_, err := tx.Exec(
			p.ctx,
			`UPDATE wb_content_cards SET deleted = $2, updated_timestamp = $3
				WHERE nm_id=$1`,
			id, true, time.Now().UTC().Format("2006-01-02 03:04:05"),
		)
		if err != nil {
			slog.Error(fmt.Sprintf("При удалении карточки %d возникла ошибка %s", id, err.Error()))
			return err
		}

	}

	slog.Info(fmt.Sprintf("Удалено %d карточек в БД", len(ids)))

	return nil
}

// syncMarketplaceStocks синхронизирует остатки полученные с api в БД
func (p *pClinet) syncMarketplaceStocks(mps *marketplaceStocks, skusRows []*contentSkusTable) error {
	tx, err := p.pool.Begin(p.ctx)
	if err != nil {
		slog.Error(fmt.Sprintf("При создании транзакции произошла ошибка %s", err.Error()))
		return err
	}

	defer tx.Rollback(p.ctx)

	for _, row := range skusRows {
		if _, ok := mps.stocks[row.Sku]; ok {
			if err := pdb.upsertMarketplaceStocks(tx, row.Sku, row.NmID, mps.getStock(row.Sku)); err != nil {
				return err
			}
		} else {
			slog.Debug(fmt.Sprintf("Для баркода %s карточки %d остаток не найден", row.Sku, row.NmID))
		}
	}

	if err := tx.Commit(pdb.ctx); err != nil {
		slog.Error(fmt.Sprintf("При коммите изменений в БД произошла ошибка %s", err.Error()))
		return err
	}
	slog.Info(fmt.Sprintf("Остатки по складам продавца успешно синхронизировны"))

	return nil
}

// upsertMarketplaceStocks обновляет запись остатков по маркетплейсу в БД
func (p *pClinet) upsertMarketplaceStocks(tx pgx.Tx, sku string, nmID uint32, ammount uint32) error {
	_, err := tx.Exec(
		p.ctx,
		`INSERT INTO wb_marketplace_stocks (sku, nm_id, amount, updated_timestamp)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (sku) DO UPDATE
				SET amount = $3, updated_timestamp = $4`,
		sku, nmID, ammount, time.Now().UTC().Format("2006-01-02 03:04:05"),
	)
	if err != nil {
		slog.Error(fmt.Sprintf("При записи остатка карточки %d (баркод %s) в базу данных возникла ошибка %s", nmID, sku, err.Error()))
		return err
	}

	return nil
}

// getSkusMarketplaceStocksTable получает массив sku из таблицы карточек
func (p *pClinet) getSkusMarketplaceStocksTable() ([]string, error) {
	rows, err := p.pool.Query(
		p.ctx, "SELECT sku FROM wb_marketplace_stocks",
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	skus, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	return skus, nil
}
