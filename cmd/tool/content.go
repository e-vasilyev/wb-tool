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

// contentCards описывает карточки товара
type contentCards struct {
	cards   []*contentCard
	trashed bool
}

// count возвращает количество карточек
func (cs *contentCards) count() int {
	return len(cs.cards)
}

// newCards создает новый список карточек полученных из api
func newCards(wbcs *wbapi.ContentCards, trashed bool) *contentCards {
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

	return &contentCards{
		cards:   cards,
		trashed: trashed,
	}
}

// contentSync синхронизирует карточки с БД
func contentSync(wbClient *wbapi.Client, job gocron.Job) {
	defer slog.Info(fmt.Sprintf("Следующий запуск задачи '%s' в %s", job.GetName(), job.NextRun()))

	// Синнхронизация корзины
	wbCards, err := wbClient.GetCardsTrash()
	if err != nil {
		slog.Error(fmt.Sprintf("При получении карточек произошла ошибка %s", err.Error()))
		return
	}
	trashedCards := newCards(wbCards, true)
	slog.Info(fmt.Sprintf("Получено %d карточек корзины", trashedCards.count()))

	if err := pdb.syncContentCards(trashedCards); err != nil {
		slog.Error(fmt.Sprintf("При сохранении карточек в БД произошла ошибка %s", err.Error()))
		return
	}

	// Синхронизация карточек
	wbCards, err = wbClient.GetCards()
	if err != nil {
		slog.Error(fmt.Sprintf("При получении карточек произошла ошибка %s", err.Error()))
		return
	}
	cards := newCards(wbCards, false)
	slog.Info(fmt.Sprintf("Получено %d карточек", cards.count()))

	if err := pdb.syncContentCards(cards); err != nil {
		slog.Error(fmt.Sprintf("При сохранении карточек в БД произошла ошибка %s", err.Error()))
		return
	}
}

// getNmIDsForDelete получает список nmID для удаления
func (cs *contentCards) getNmIDsForDelete() ([]uint32, error) {
	var ids []uint32
	var err error
	var res []uint32

	if cs.trashed {
		ids, err = pdb.getTrashedNmIDsConentCardsTable()
		slog.Info(fmt.Sprintf("Получено %d карточек карзины из БД", len(ids)))
	} else {
		ids, err = pdb.getNmIDsConentCardsTable()
		slog.Info(fmt.Sprintf("Получено %d карточек из БД", len(ids)))
	}

	if err != nil {
		return []uint32{}, err
	}

	diff := make(map[uint32]struct{}, cs.count())
	for _, c := range cs.cards {
		diff[c.nmID] = struct{}{}
	}

	for _, id := range ids {
		if _, ok := diff[id]; !ok {
			res = append(res, id)
			slog.Info(fmt.Sprintf("Карточка %d пристутвует в БД, но отсутвует в магазине", id))
		}
	}
	return res, nil
}

// checkingTimeSpentInTrash проверяет как долго карточки хранятся в коризне.
// Если карточка хранится слишком долго, то автоматически восстанавливается и обратно помещяется в корзину.
// При этом проверяется остаток, если остаток не 0 то карточка остается в корзине
func checkingTimeSpentInTrash(wbClient *wbapi.Client, job gocron.Job) {
	defer slog.Info(fmt.Sprintf("Следующий запуск задачи '%s' в %s", job.GetName(), job.NextRun()))

	maxDays := config.GetInt("max_days_in_trash")
	slog.Info(fmt.Sprintf("Запущен поиск карточек в карзине старше %d дней", maxDays))

	nmIDs, err := pdb.getContentCardsForRecoverToExpire(maxDays)
	if err != nil {
		slog.Error(fmt.Sprintf("При получении карточек в БД для передобавления в корзниу произошла ошибка %s", err.Error()))
		return
	}
	slog.Debug(fmt.Sprintf("Найдено %d карточек в БД старше %d дней", len(nmIDs), maxDays))

	for _, nmID := range nmIDs {
		if err := recoverAndMoveToTrash(wbClient, nmID); err == nil {
			slog.Info(fmt.Sprintf("Карточка %d передобавлена в корзину", nmID))
		}
	}

}

// recoverAndMoveToTrash востанавливает указанную карточку из корзины и возвращает обратно
func recoverAndMoveToTrash(wbClient *wbapi.Client, nmID uint32) error {
	tx, err := pdb.pool.Begin(pdb.ctx)
	if err != nil {
		slog.Error(fmt.Sprintf("При создании транзакции произошла ошибка %s", err.Error()))
		return err
	}

	defer tx.Rollback(pdb.ctx)

	if err := pdb.recoverCard(tx, nmID); err != nil {
		return err
	}

	nmIDs := []uint32{nmID}
	if err := wbClient.RecoverCards(nmIDs); err != nil {
		slog.Error(fmt.Sprintf("При востановлении карточки %d возникла ошибка %s", nmID, err.Error()))
		return err
	}

	if err := pdb.moveToTrash(tx, nmID); err != nil {
		return err
	}

	if err := wbClient.MoveToTrash(nmIDs); err != nil {
		slog.Error(fmt.Sprintf("При переносе карточки %d в корзину возникла ошибка %s", nmID, err.Error()))
		return err
	}

	if err := tx.Commit(pdb.ctx); err != nil {
		slog.Error(fmt.Sprintf("При коммите изменений в БД произошла ошибка %s", err.Error()))
		return err
	}

	return nil
}
