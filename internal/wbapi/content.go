package wbapi

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	contentPathPing             string = "ping"
	contentPathCards            string = "content/v2/get/cards/list"
	contentPathCardsTrash       string = "content/v2/get/cards/trash"
	contentPathCardsDeleteTrash string = "content/v2/cards/delete/trash"
	contentPathCardsRecover     string = "content/v2/cards/recover"
	contentRequestLimit         uint   = 100
)

// contentRequestTicker канал контролирующй количество отправленных запросов в минуту.
// 60 запросов в минуту в разделе контента (раз в секунду)
var contentRequestTicker <-chan time.Time = time.NewTicker(time.Second).C

// ContentCardCursor описывает блок size в карточке товара
type ContentCardCursor struct {
	NmID      uint32 `json:"nmID,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	TrashedAt string `json:"trashedAt,omitempty"`
	Total     uint   `json:"total,omitempty"`
}

// ContentCardSize описывает блок size в карточке товара
type ContentCardSize struct {
	Skus []string `json:"skus"`
}

// ContentCard описывает карточку в разделе content
type ContentCard struct {
	NmID        uint32            `json:"nmID"`
	ImtID       uint32            `json:"imtID,omitempty"`
	VendorCode  string            `json:"vendorCode"`
	SubjectID   uint32            `json:"subjectID"`
	SubjectName string            `json:"subjectName"`
	Brand       string            `json:"brand,omitempty"`
	Title       string            `json:"title,omitempty"`
	Sizes       []ContentCardSize `json:"sizes"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt,omitempty"`
	TrashedAt   string            `json:"trashedAt,omitempty"`
}

// ContentCards описывает карточки в разделе content
type ContentCards struct {
	Cards  []ContentCard     `json:"cards"`
	Cursor ContentCardCursor `json:"cursor"`
}

// contentCursorRequest описывает блок cursor в запросе
type contentCursorRequest struct {
	ContentCardCursor
	Limit uint `json:"limit"`
}

// contentFilterRequest описывает блок filter в запросе
type contentFilterRequest struct {
	WithPhoto int `json:"withPhoto"`
}

// contentSettingsRequest описывает блок settings в запросе
type contentSettingsRequest struct {
	Cursor contentCursorRequest `json:"cursor,omitempty"`
	Filter contentFilterRequest `json:"filter"`
}

// contentRequest описывает запрос к контенту
type contentRequest struct {
	Settings contentSettingsRequest `json:"settings"`
}

// contentDeleteTrashRequest описывает запрос для перемещения карточки в корзину
type contentDeleteTrashRequest struct {
	NmIDs []uint32 `json:"nmIDs"`
}

// contentRecover описывает запрос для восстановления карточки из корзины
type contentRecover contentDeleteTrashRequest

// GetCardsTrash получает все карточки из корзины
// Так как получить за раз можно не все карточки, выполняются несколько запросов к
// полученю карточек
func (c *Client) GetCardsTrash() (*ContentCards, error) {
	c.logger.Debug("Получение карточек из корзины")

	contentCards := &ContentCards{
		Cards:  []ContentCard{},
		Cursor: ContentCardCursor{},
	}
	url := fmt.Sprintf("%s/%s", c.baseURL.content, contentPathCardsTrash)

	body := &contentRequest{
		Settings: contentSettingsRequest{
			Cursor: contentCursorRequest{Limit: contentRequestLimit},
			Filter: contentFilterRequest{WithPhoto: -1},
		},
	}

	for {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		contentCardsPage, err := c.getCards(url, jsonBody)

		if err != nil {
			return nil, err
		}

		contentCards.Cards = append(contentCards.Cards, contentCardsPage.Cards...)
		contentCards.Cursor = contentCardsPage.Cursor

		body.Settings.Cursor.NmID = contentCardsPage.Cursor.NmID
		body.Settings.Cursor.TrashedAt = contentCardsPage.Cursor.TrashedAt

		if contentCardsPage.Cursor.Total < contentRequestLimit {
			break
		}
	}

	return contentCards, nil
}

// GetCards получает все карточки, кроме карточек в корзине
// Так как получить за раз можно не все карточки, выполняются несколько запросов к
// полученю карточек
func (c *Client) GetCards() (*ContentCards, error) {
	c.logger.Debug("Получение карточек")

	contentCards := &ContentCards{
		Cards:  []ContentCard{},
		Cursor: ContentCardCursor{},
	}
	url := fmt.Sprintf("%s/%s?locale=ru", c.baseURL.content, contentPathCards)

	body := &contentRequest{
		Settings: contentSettingsRequest{
			Cursor: contentCursorRequest{Limit: contentRequestLimit},
			Filter: contentFilterRequest{WithPhoto: -1},
		},
	}

	for {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		contentCardsPage, err := c.getCards(url, jsonBody)

		if err != nil {
			return nil, err
		}

		contentCards.Cards = append(contentCards.Cards, contentCardsPage.Cards...)
		contentCards.Cursor = contentCardsPage.Cursor

		body.Settings.Cursor.NmID = contentCardsPage.Cursor.NmID
		body.Settings.Cursor.UpdatedAt = contentCardsPage.Cursor.UpdatedAt

		if contentCardsPage.Cursor.Total < contentRequestLimit {
			break
		}
	}

	return contentCards, nil
}

// getCards получает карточки. Запрос выдаст ограниченное количество карточек
// в зависимости от jsonBody
func (c *Client) getCards(url string, jsonBody []byte) (*ContentCards, error) {
	contentCards := &ContentCards{}

	res, err := c.postRequest(url, jsonBody, contentRequestTicker)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if err := respCodeCheck(res); err != nil {
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(contentCards)

	return contentCards, err
}

// MoveToTrash переносит карточки в коризину
// nmIDs ограничен 1000 позициями
func (c *Client) MoveToTrash(nmIDs []uint32) error {
	if len(nmIDs) > 1000 {
		return fmt.Errorf("количество карточек в запросе не должно быть больше 1000. Текущее значение: %d", len(nmIDs))
	}

	url := fmt.Sprintf("%s/%s", c.baseURL.content, contentPathCardsDeleteTrash)

	body := &contentDeleteTrashRequest{
		NmIDs: nmIDs,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	res, err := c.postRequest(url, jsonBody, contentRequestTicker)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err := respCodeCheck(res); err != nil {
		return err
	}

	return nil
}

// RecoverCards восстанавливает карточки из корзины
// nmIDs ограничен 1000 позициями
func (c *Client) RecoverCards(nmIDs []uint32) error {
	if len(nmIDs) > 1000 {
		return fmt.Errorf("количество карточек в запросе не должно быть больше 1000. Текущее значение: %d", len(nmIDs))
	}

	url := fmt.Sprintf("%s/%s", c.baseURL.content, contentPathCardsRecover)

	body := &contentRecover{
		NmIDs: nmIDs,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	res, err := c.postRequest(url, jsonBody, contentRequestTicker)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err := respCodeCheck(res); err != nil {
		return err
	}

	return nil
}
