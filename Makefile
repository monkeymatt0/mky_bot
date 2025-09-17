# Makefile per Trading Bot Database

.PHONY: Helper for the project

# Help
help: ## Mostra questo help
	@echo "Comandi disponibili:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build
build: ## Compila il progetto
	go build -o bin/trading-bot ./cmd/main.go

#
# Dependencies
deps: ## Installa le dipendenze
	go mod tidy
	go mod download

# Run
run: ## Esegue l'applicazione
	go run ./cmd/main.go
