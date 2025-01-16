package wbapi

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client описывает подключение к API WB
type Client struct {
	token   string
	baseURL *ClientBaseURL
	logger  *slog.Logger
}

// ClientOptions интерфейс дополнительных опций клиента
type ClientOptions interface {
	apply(c *Client)
}

type optionFunc func(c *Client)

func (fn optionFunc) apply(c *Client) {
	fn(c)
}

// ClientBaseURL список базовых URL до различных API
type ClientBaseURL struct {
	content     string
	marketplace string
}

// SetClientBaseURL задает базовые URL
func SetClientBaseURL(clientBaseURL *ClientBaseURL) ClientOptions {
	return optionFunc(func(c *Client) {
		c.baseURL = clientBaseURL
	})
}

// SetClientLogger задает настройки логгера
func SetClientLogger(logger *slog.Logger) ClientOptions {
	return optionFunc(func(c *Client) {
		c.logger = logger
	})
}

// defaultClientBaseURL значение по умолчанию базовых URL
var defaultClientBaseURL *ClientBaseURL = &ClientBaseURL{
	content:     "https://content-api.wildberries.ru",
	marketplace: "https://marketplace-api.wildberries.ru",
}

// NewClient создает клиента подключения
func NewClient(token string) *Client {
	return NewClientWithOptions(token)
}

// NewClientWithOptions создает клиента подключения c указанными опциями
func NewClientWithOptions(token string, opts ...ClientOptions) *Client {
	client := &Client{
		token:   token,
		baseURL: defaultClientBaseURL,
		logger:  slog.New(&slog.TextHandler{}),
	}

	for _, opt := range opts {
		opt.apply(client)
	}

	// Запуск фоновой функции по понтролю количетсва запросов в минуту
	go func() {
		client.logger.Debug("Запуск функции контроля запросов в минуту")
		for {
			select {
			case <-time.NewTicker(time.Minute).C:
				c := len(contentRequestCount)
				client.logger.Debug(fmt.Sprintf("Количество запросов в минуту к контенту: %d", c))
				for i := 0; i < c; i++ {
					<-contentRequestCount
				}
			case <-time.NewTicker(time.Minute).C:
				c := len(marketplaceRequestCount)
				client.logger.Debug(fmt.Sprintf("Количество запросов в минуту к маркетплейсу: %d", c))
				for i := 0; i < c; i++ {
					<-marketplaceRequestCount
				}
			}
		}
	}()

	return client
}

// Ping проверяет доступность API WB
func (c Client) Ping() error {
	if err := c.contentPing(); err != nil {
		return err
	}

	if err := c.marketplacePing(); err != nil {
		return err
	}

	return nil
}

// contentPing проверяет доступность API Content
func (c Client) contentPing() error {
	c.logger.Debug("Проверка достпности API контента")

	url := fmt.Sprintf("%s/%s", c.baseURL.content, contentPathPing)

	return c.requestPing(url, contentRequestCount)
}

// marketplacePing проверяет доступность API Marketplace
func (c Client) marketplacePing() error {
	c.logger.Debug("Проверка достпности API маркетплейса")

	url := fmt.Sprintf("%s/%s", c.baseURL.marketplace, marketplacePathPing)

	return c.requestPing(url, marketplaceRequestCount)
}

// requestPing делает запрос ping к указанному ресурсу
func (c Client) requestPing(url string, ch chan uint8) error {
	res, err := c.getRequest(url, ch)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)

	if res.StatusCode != 200 {
		err = fmt.Errorf("code: %d, body: %s", res.StatusCode, buf.String())
	}

	return err
}

// postRequest делает POST запрос обогащенный заголовками
// В ответе получаем http.Response без обработки
func (c Client) postRequest(uri string, data []byte, ch chan uint8) (*http.Response, error) {
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	return c.httpRequest(req, ch)
}

// getRequest делает Get запрос обогащенный заголовками
// В ответе получаем http.Response без обработки
func (c Client) getRequest(url string, ch chan uint8) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.httpRequest(req, ch)
}

// httpRequest делает запрос к API.
// Тип запроса определяется во входящем параметре.
// Если запрос возвращает code 429, то запрос повторяется через некоторое время
func (c Client) httpRequest(req *http.Request, ch chan uint8) (*http.Response, error) {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", c.token)

	client := &http.Client{}

	c.logger.Debug(fmt.Sprintf("Запрос к %s%s", req.URL.Host, req.URL.Path))

	var delay time.Duration = 30
	for {
		ch <- 1

		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if res.StatusCode != 429 {
			return res, nil
		}

		c.logger.Debug(fmt.Sprintf("Получен статус ответа %s. Ожидание %d секунд", res.Status, delay))
		time.Sleep(delay * time.Second)
		delay += 30
	}
}

// respCodeCheck проверяет HTTP ответ на коды отличные от 200
func respCodeCheck(res *http.Response) error {
	var buf bytes.Buffer
	tee := io.TeeReader(res.Body, &buf)

	if res.StatusCode != 200 {
		return fmt.Errorf("code: %d, body: %s", res.StatusCode, tee)
	}

	return nil
}
