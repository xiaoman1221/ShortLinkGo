# syntax=docker/dockerfile:1
# 多阶段构建：先构建 Vue 前端，再编译 Go 后端，最后打包为精简运行镜像。

# ---------- 基础镜像（可用构建参数覆盖，便于切换国内镜像源） ----------
ARG BASE_NODE_IMAGE=node:20-alpine
ARG BASE_GO_IMAGE=golang:1.27-alpine
ARG BASE_RUNTIME_IMAGE=alpine:3.20

# ---------- 阶段一：构建前端 ----------
FROM ${BASE_NODE_IMAGE} AS web-builder
ARG NPM_REGISTRY=https://registry.npmmirror.com
WORKDIR /app/web
COPY web/package*.json ./
RUN npm config set registry ${NPM_REGISTRY} \
    && npm install
COPY web/ ./
RUN npm run build

# ---------- 阶段二：构建后端 ----------
FROM ${BASE_GO_IMAGE} AS builder
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    CGO_ENABLED=0
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags "-s -w" -o /out/shortlinkgo ./server

# ---------- 阶段三：运行镜像 ----------
FROM ${BASE_RUNTIME_IMAGE}
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /out/shortlinkgo /app/shortlinkgo
COPY --from=web-builder /app/web/dist /app/web/dist
COPY docs/ /app/docs/
ENV GIN_MODE=release \
    PORT=8080 \
    DB_PATH=/app/data/data.db \
    HOST=http://localhost:8080
RUN mkdir -p /app/data && chown -R app:app /app
USER app
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/app/shortlinkgo"]
