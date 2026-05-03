.PHONY: build test run clean docker-build docker-up docker-down

build:
	go build -o javapi ./cmd/api

test:
	go test ./... -v -count=1

run:
	go run ./cmd/api

clean:
	rm -f javapi

docker-build:
	docker build -t javapi .

docker-up:
	docker compose up -d

docker-down:
	docker compose down
