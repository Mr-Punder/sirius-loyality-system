package telegrambot

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
)

// APIClient представляет клиент для работы с API сервера
type APIClient struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
	logger     logger.Logger
}

// NewAPIClient создает новый клиент для работы с API
func NewAPIClient(baseURL, apiToken string, logger logger.Logger) *APIClient {
	return &APIClient{
		baseURL:  baseURL,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// Get выполняет GET-запрос к API
func (c *APIClient) Get(path string, params map[string]string) ([]byte, error) {
	// Формируем URL
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга URL: %w", err)
	}
	reqURL.Path = path

	// Добавляем параметры запроса
	if params != nil {
		query := reqURL.Query()
		for key, value := range params {
			query.Set(key, value)
		}
		reqURL.RawQuery = query.Encode()
	}

	// Создаем запрос
	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Добавляем заголовок авторизации
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	// Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка API (%d): %s", resp.StatusCode, string(body))
	}

	// Проверяем, сжат ли ответ
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания gzip-ридера: %w", err)
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	// Читаем ответ
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	return body, nil
}

// Post выполняет POST-запрос к API
func (c *APIClient) Post(path string, data interface{}) ([]byte, error) {
	// Формируем URL
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга URL: %w", err)
	}
	reqURL.Path = path

	// Кодируем данные в JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("ошибка кодирования JSON: %w", err)
	}

	// Создаем запрос
	req, err := http.NewRequest(http.MethodPost, reqURL.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	// Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка API (%d): %s", resp.StatusCode, string(body))
	}

	// Проверяем, сжат ли ответ
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания gzip-ридера: %w", err)
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	// Читаем ответ
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	return body, nil
}

// Delete выполняет DELETE-запрос к API
func (c *APIClient) Delete(path string) ([]byte, error) {
	// Формируем URL
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга URL: %w", err)
	}
	reqURL.Path = path

	// Создаем запрос
	req, err := http.NewRequest(http.MethodDelete, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Добавляем заголовок авторизации
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	// Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка API (%d): %s", resp.StatusCode, string(body))
	}

	// Проверяем, сжат ли ответ
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания gzip-ридера: %w", err)
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	// Читаем ответ
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	return body, nil
}

// PostFile выполняет POST-запрос с файлом (multipart/form-data)
func (c *APIClient) PostFile(path string, file io.Reader, filename string) error {
	// Формируем URL
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("ошибка парсинга URL: %w", err)
	}
	reqURL.Path = path

	// Создаем multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Добавляем файл
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("ошибка создания form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("ошибка копирования файла: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("ошибка закрытия writer: %w", err)
	}

	// Создаем запрос
	req, err := http.NewRequest(http.MethodPost, reqURL.String(), body)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	// Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка API (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Patch выполняет PATCH-запрос к API
func (c *APIClient) Patch(path string, data interface{}) ([]byte, error) {
	// Формируем URL
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга URL: %w", err)
	}
	reqURL.Path = path

	// Кодируем данные в JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("ошибка кодирования JSON: %w", err)
	}

	// Создаем запрос
	req, err := http.NewRequest(http.MethodPatch, reqURL.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	// Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка API (%d): %s", resp.StatusCode, string(body))
	}

	// Проверяем, сжат ли ответ
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания gzip-ридера: %w", err)
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	// Читаем ответ
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	return body, nil
}
