.PHONY: build build-panel build-agent test run docker run-docker clean

build: build-panel build-agent

build-panel:
	go build -o bin/panel ./cmd/panel

build-agent:
	go build -o bin/agent ./cmd/agent

test:
	go test -race -v ./...

run:
	go run ./cmd/panel

run-agent:
	go run ./cmd/agent

docker:
	docker build -t vpn-panel:latest .

docker-agent:
	docker build -f Dockerfile.agent -t vpn-agent:latest .

clean:
	rm -rf bin/
