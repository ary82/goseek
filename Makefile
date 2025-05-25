all: build

build:
	@echo "building go binary..."
	@go build -o main ./cmd/ssh/main.go

run: build
	@echo "running..."
	@./main
