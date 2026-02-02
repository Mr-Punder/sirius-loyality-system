package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/MrPunder/sirius-loyality-system/internal/admin"
	"github.com/MrPunder/sirius-loyality-system/internal/config"
)

func main() {
	// Парсим аргументы командной строки
	password := flag.String("password", "", "Новый пароль администратора (минимум 8 символов)")
	flag.Parse()

	if *password == "" {
		fmt.Println("Ошибка: пароль не указан")
		fmt.Println("Использование: setadminpassword -password=НОВЫЙ_ПАРОЛЬ")
		os.Exit(1)
	}

	conf, err := config.LoadConfig("")
	if err != nil {
		fmt.Printf("Ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	// Создаем менеджер паролей
	passwordMgr := admin.NewPasswordManager(conf.Storage.DataPath)

	// Устанавливаем пароль
	err = passwordMgr.SetPassword(*password)
	if err != nil {
		fmt.Printf("Ошибка установки пароля: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Пароль администратора успешно установлен")
}
