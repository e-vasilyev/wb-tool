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

// syncContentCards синхронизирует данные получение с api в БД
func (p *pClinet) syncContentCards(cs *[]contentCard) error {
	tx, err := p.pool.Begin(p.ctx)
	if err != nil {
		slog.Error(fmt.Sprintf("Ошибка при создании транзакции: %s", err.Error()))
		return err
	}

	defer tx.Rollback(p.ctx)

	for _, card := range *cs {
		if card.isTrashed() {
			err = p.upsertContentTrashedCard(tx, card)
		} else {
			err = p.upsertContentCard(tx, card)
		}

		if err != nil {
			return err
		}
	}

	if err := tx.Commit(pdb.ctx); err != nil {
		slog.Error(fmt.Sprintf("Ошибка при коммите изменений: %s", err.Error()))
		return err
	}
	slog.Info(fmt.Sprintf("Карточки успешно добавлены в БД"))

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
		slog.Error(fmt.Sprintf("Ошибка записи карточки из корзины в базу данных: %s", err.Error()))
		return err
	}

	return nil
}

// upsertContentCard добавляет запись карторчки в БД
func (p *pClinet) upsertContentCard(tx pgx.Tx, card contentCard) error {
	_, err := tx.Exec(
		p.ctx,
		`INSERT INTO wb_content_cards (nm_id, imtID, vendor_code, subject_id, subject_name, brand, title, trashed, updated_timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (nm_id) DO UPDATE
				SET imtID = $2, vendor_code = $3, subject_id = $4, subject_name = $5, 
					brand = $6, title = $7, trashed = $8, updated_timestamp = $9`,
		card.nmID, card.imtID, card.vendorCode, card.subjectID,
		card.subjectName, card.brand, card.title, false,
		time.Now().UTC().Format("2006-01-02 03:04:05"),
	)
	if err != nil {
		slog.Error(fmt.Sprintf("Ошибка записи карточки в базу данных: %s", err.Error()))
		return err
	}

	return nil
}
