.PHONY: all build web run clean docker-up docker-down

all: build

# 一键构建（前端 + 后端）
build: web
	go build -trimpath -ldflags "-s -w" -o shortlinkgo ./server

# 构建前端
web:
	cd web && npm install && npm run build

# 本地运行（开发）
run:
	go run ./server

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

clean:
	rm -f shortlinkgo shortlinkgo.exe
	rm -rf web/dist
