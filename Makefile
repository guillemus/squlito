build:
	go build ./...

.PHONY: install
install:
	mkdir -p "$(HOME)/.local/bin"
	go build -o "$(HOME)/.local/bin/squlito" ./cmd/squlito

run:
	go run cmd/squlito/main.go data/seed.db

seed:
	go run cmd/seed/main.go

test:
	go test ./...
