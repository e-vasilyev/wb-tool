package wbapi

import (
	"encoding/json"
	"fmt"
)

const (
	contentPathPing          string = "ping"
	contentPathCardsTrash    string = "content/v2/get/cards/trash"
	contentRequestLimit      uint   = 100
	contentRequestsPerMinute uint8  = 60 // Количество разрешенных запросов в минуту
)

// contentRequestCount канал содержащий количество отправленных запросов
// в разделе контента.
var contentRequestCount chan uint8 = make(chan uint8, contentRequestsPerMinute)

// ContentCardCursor описывает блок size в карточке товара
type ContentCardCursor struct {
	TrashedAt string `json:"trashedAt,omitempty"`
	NmID      uint   `json:"nmID,omitempty"`
	Total     uint   `json:"total,omitempty"`
}

// ContentCardSize описывает блок size в карточке товара
type ContentCardSize struct {
	Skus []string `json:"skus"`
}

// ContentCard описывает карточку в разделе content
type ContentCard struct {
	NmID        uint              `json:"nmID"`
	VendorCode  string            `json:"vendorCode"`
	SubjectID   uint              `json:"subjectID"`
	SubjectName string            `json:"subjectName"`
	Sizes       []ContentCardSize `json:"sizes"`
	CreatedAt   string            `json:"createdAt"`
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

// getCards получает карточки. Запрос выдаст ограниченное количество карточек
// в зависимости от jsonBody
func (c *Client) getCards(url string, jsonBody []byte) (*ContentCards, error) {
	contentCards := &ContentCards{}

	res, err := c.postRequest(url, jsonBody, contentRequestCount)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := respCodeCheck(res); err != nil {
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(contentCards)
	if err != nil {
		return nil, err
	}

	return contentCards, nil
}
