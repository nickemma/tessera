.PHONY: run build test fmt compose-up compose-down e2e chaos bench real-model

run:
	go run ./cmd/gateway

build:
	go build ./...

test:
	go test -race ./...

fmt:
	gofmt -w cmd internal

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down

e2e:
	sh scripts/e2e.sh

chaos:
	sh scripts/chaos.sh

bench:
	go run ./bench -key "$(TESSERA_API_KEY)" $(BENCH_FLAGS)

real-model:
	sh scripts/real-model.sh
