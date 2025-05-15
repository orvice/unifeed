.PHONY: build test clean

# 默认目标
all: build

# 构建项目
build:
	@echo "Building project..."
	go build -o bin/unifeed ./cmd/main.go

# 运行测试
test:
	@echo "Running tests..."
	go test -v ./...

# 清理构建产物
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/ 