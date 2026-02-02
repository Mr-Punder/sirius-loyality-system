.PHONY: run-server run-userbot run-adminbot build-linux clean

run-server:
	@if [ ! -f local.yaml ]; then \
		echo "⚠ Файл local.yaml не найден"; \
		exit 1; \
	fi
	@mkdir -p logs
	@cd server && CONFIG_PATH=../local.yaml go run ./cmd/loyalityserver/main.go

run-userbot:
	@if [ ! -f local.yaml ]; then \
		echo "⚠ Файл local.yaml не найден"; \
		exit 1; \
	fi
	@if [ ! -f config/token.txt ]; then \
		echo "⚠ Файл config/token.txt не найден"; \
		exit 1; \
	fi
	@cd server && CONFIG_PATH=../local.yaml \
		TOKEN_PATH=../config/token.txt \
		API_TOKEN_PATH=../config/api_token.txt \
		SERVER_URL=http://localhost:8080 \
		go run ./cmd/telegrambot/user/main.go

run-adminbot:
	@if [ ! -f local.yaml ]; then \
		echo "⚠ Файл local.yaml не найден"; \
		exit 1; \
	fi
	@if [ ! -f config/admin_token.txt ]; then \
		echo "⚠ Файл config/admin_token.txt не найден"; \
		exit 1; \
	fi
	@if [ ! -f config/admins.json ]; then \
		echo "⚠ Файл config/admins.json не найден"; \
		exit 1; \
	fi
	@cd server && CONFIG_PATH=../local.yaml \
		TOKEN_PATH=../config/admin_token.txt \
		API_TOKEN_PATH=../config/api_token.txt \
		ADMINS_PATH=../config/admins.json \
		SERVER_URL=http://localhost:8080 \
		go run ./cmd/telegrambot/admin/main.go

build-linux:
	@cd server && go mod download
	@cd server && GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o loyalityserver ./cmd/loyalityserver
	@cd server && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o userbot ./cmd/telegrambot/user
	@cd server && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o adminbot ./cmd/telegrambot/admin

clean:
	@rm -f server/loyalityserver server/userbot server/adminbot
