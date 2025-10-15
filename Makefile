.PHONY: build deps lint test docker integration-test docs

BINARY_NAME=islectl

deps:
	go get .
	go mod tidy

build: deps
	go build -o $(BINARY_NAME) .

lint:
	go fmt ./...
	golangci-lint run

	@if command -v json5 > /dev/null 2>&1; then \
		echo "Running json5 validation on renovate.json5"; \
		json5 --validate renovate.json5 > /dev/null; \
	else \
		echo "json5 not found, skipping renovate validation"; \
	fi

test: build
	go test -v -race ./...

docker:
	docker build -t ghcr.io/islandora-devops/islectl:ci ./ci

integration-test: docker
	./ci/test.sh

docs:
	docker build -t $(BINARY_NAME)-docs:latest docs
	@docker stop $(BINARY_NAME)-docs 2>/dev/null || true
	@PORT=8080; \
	while lsof -Pi :$$PORT -sTCP:LISTEN -t >/dev/null 2>&1; do \
		PORT=$$((PORT + 1)); \
	done; \
	echo "Starting documentation server at http://localhost:$$PORT"; \
	docker run -d --rm --name $(BINARY_NAME)-docs -p $$PORT:80 $(BINARY_NAME)-docs:latest > /dev/null

