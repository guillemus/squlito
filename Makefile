build:
	go build ./...

run:
	go run cmd/squlito/main.go data/seed.db

seed:
	go run cmd/seed/main.go

test:
	go test ./...
