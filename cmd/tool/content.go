package main

import (
	"fmt"
	"log/slog"

	"github.com/e-vasilyev/wb-tool/internal/wbapi"
	"github.com/go-co-op/gocron"
)

// contentCard описывает карточку товара
type contentCard struct {
	nmID        uint32
	imtID       uint32
	vendorCode  string
	subjectID   uint32
	subjectName string
	brand       string
	title       string
	skus        []string
	createdAt   string
	updatedAt   string
	trashedAt   string
}

// isTrashed проверяет в корзине ли находится карточка
func (c contentCard) isTrashed() bool {
	if c.trashedAt != "" {
		return true
	}

	return false
}

// newCards создает новый список карточек полученных из api
func newCards(wbcs *wbapi.ContentCards) []*contentCard {
	var cards []*contentCard

	for _, c := range wbcs.Cards {
		var skus []string

		for _, s := range c.Sizes {
			skus = append(skus, s.Skus...)
		}

		card := &contentCard{
			nmID:        c.NmID,
			imtID:       c.ImtID,
			vendorCode:  c.VendorCode,
			subjectID:   c.SubjectID,
			subjectName: c.SubjectName,
			brand:       c.Brand,
			title:       c.Title,
			skus:        skus,
			createdAt:   c.CreatedAt,
			updatedAt:   c.UpdatedAt,
			trashedAt:   c.TrashedAt,
		}
		cards = append(cards, card)
	}
	return cards
}

// contentSync синхронизирует карточки с БД
func contentSync(wbClient *wbapi.Client, job gocron.Job) {
	// Синнхронизация корзины
	wbCards, err := wbClient.GetCardsTrash()
	if err != nil {
		slog.Error(fmt.Sprintf("При получении карточек произошла ошибка %s", err.Error()))
		return
	}
	cards := newCards(wbCards)
	slog.Info(fmt.Sprintf("Получено %d карточек в корзине", len(cards)))

	if err := pdb.syncContentCards(cards); err != nil {
		slog.Error(fmt.Sprintf("При сохранении карточек в БД произошла ошибка %s", err.Error()))
		return
	}

	// Синхронизация карточек
	wbCards, err = wbClient.GetCards()
	if err != nil {
		slog.Error(fmt.Sprintf("При получении карточек произошла ошибка %s", err.Error()))
		return
	}
	cards = newCards(wbCards)
	slog.Info(fmt.Sprintf("Получено %d карточек", len(cards)))

	if err := pdb.syncContentCards(cards); err != nil {
		slog.Error(fmt.Sprintf("При сохранении карточек в БД произошла ошибка %s", err.Error()))
		return
	}

	slog.Info(fmt.Sprintf("Следующий запуск задачи %s в %s", job.GetName(), job.NextRun()))
}
