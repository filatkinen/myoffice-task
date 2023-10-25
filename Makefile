BIN := "./build/cliurl"
FILE = $(URL)

build:
	go build -v -o $(BIN) -ldflags "$(LDFLAGS)" ./cmd

run: build
	$(BIN) -f $(FILE)

test:
	go test -race -v --count=1 ./...



build-img:
	docker-compose -f deployment/docker-compose.yaml build

run-img: build-img
	docker-compose -f deployment/docker-compose.yaml up -d


up: run-img

docker-run:run-img
	docker exec -it urlcli  make FILE=url.txt run

docker-test:run-img
	docker exec -it urlcli  go test -race -v --count=1 ./...

down:
	docker-compose -f deployment/docker-compose.yaml down \
		 --rmi local \
		--volumes \
		--remove-orphans \
		--timeout 5; \



install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2

lint: install-lint-deps
	golangci-lint run ./...


.PHONY: build test lint
