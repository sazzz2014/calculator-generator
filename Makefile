.PHONY: native build test verify smoke up down fmt
native:
	cargo test --locked --manifest-path native/rust/Cargo.toml
	cargo build --locked --release --manifest-path native/rust/Cargo.toml
build: native
	go build -o bin/calculator ./cmd/calculator
	go build -o bin/generator ./cmd/generator
test: native
	go test ./...
	go test -race ./...
	go vet ./...
fmt:
	gofmt -w cmd internal scripts
verify:
	docker build --target verify -t nativecalculator-verify .
	docker compose build
	go run ./scripts/smoke
smoke:
	go run ./scripts/smoke
up:
	docker compose up --build
down:
	docker compose down
