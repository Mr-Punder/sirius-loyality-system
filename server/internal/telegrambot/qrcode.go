package telegrambot

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

// GenerateQRCode генерирует QR-код в виде PNG-изображения
func GenerateQRCode(content string, size int) ([]byte, error) {
	// Создаем QR-код
	qrCode, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("ошибка кодирования QR-кода: %w", err)
	}

	// Изменяем размер QR-кода
	qrCode, err = barcode.Scale(qrCode, size, size)
	if err != nil {
		return nil, fmt.Errorf("ошибка изменения размера QR-кода: %w", err)
	}

	// Кодируем QR-код в PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, qrCode); err != nil {
		return nil, fmt.Errorf("ошибка кодирования PNG: %w", err)
	}

	return buf.Bytes(), nil
}
