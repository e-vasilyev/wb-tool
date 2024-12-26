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
func newCards(wbcs *wbapi.ContentCards) *[]contentCard {
	var cards *[]contentCard = &[]contentCard{}

	for _, c := range wbcs.Cards {
		card := &contentCard{
			nmID:        c.NmID,
			imtID:       c.ImtID,
			vendorCode:  c.VendorCode,
			subjectID:   c.SubjectID,
			subjectName: c.SubjectName,
			brand:       c.Brand,
			title:       c.Title,
			createdAt:   c.CreatedAt,
			updatedAt:   c.UpdatedAt,
			trashedAt:   c.TrashedAt,
		}
		*cards = append(*cards, *card)
	}
	return cards
}

// contentSync синхронизирует карточки с БД
func contentSync(wbClient *wbapi.Client, job gocron.Job) {
	// Синнхронизация корзины
	wbCards, err := wbClient.GetCardsTrash()
	if err != nil {
		slog.Error(fmt.Sprintf("Ошибка при получении карточек в корзине %s", err.Error()))
		return
	}
	cards := newCards(wbCards)
	slog.Info(fmt.Sprintf("Получено %d карточек в корзине", len(*cards)))

	if err := pdb.syncContentCards(cards); err != nil {
		slog.Error(fmt.Sprintf("Ошибка при сохранении карточек в БД %s", err.Error()))
		return
	}

	// Синхронизация карточек
	wbCards, err = wbClient.GetCards()
	if err != nil {
		slog.Error(fmt.Sprintf("Ошибка при получении карточек %s", err.Error()))
		return
	}
	cards = newCards(wbCards)
	slog.Info(fmt.Sprintf("Получено %d карточек", len(*cards)))

	if err := pdb.syncContentCards(cards); err != nil {
		slog.Error(fmt.Sprintf("Ошибка при сохранении карточек в БД %s", err.Error()))
		return
	}

	slog.Info(fmt.Sprintf("Следующий запуск задачи %s в %s", job.GetName(), job.NextRun()))
}
