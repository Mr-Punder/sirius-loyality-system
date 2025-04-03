package telegrambot

import (
	"fmt"
	"regexp"
)

// Регулярное выражение для проверки группы
var GroupRegex = regexp.MustCompile(`^[НHнh][1-6]$`)

// NormalizeGroupName нормализует группу (Н1-Н6, H1-H6, н1-н6, h1-h6) -> Н1-Н6
func NormalizeGroupName(group string) (string, bool) {
	if !GroupRegex.MatchString(group) {
		return "", false
	}

	// Получаем номер группы
	number := group[len(group)-1]

	// Возвращаем нормализованную группу
	return fmt.Sprintf("Н%c", number), true
}
