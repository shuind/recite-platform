# --- 阶段 1: 构建阶段 ---
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker/main.go

# --- 阶段 2: 运行阶段 ---
FROM alpine:latest

# 安装 tzdata 以提供时区数据库（避免 time.LoadLocation 在容器中失败）
RUN apk add --no-cache tzdata

# 设置时区环境变量（可选，但推荐）
ENV TZ=Asia/Shanghai

# 将时区文件复制为系统时区（某些库读取 /etc/localtime）
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /root/

COPY --from=builder /server .
COPY --from=builder /worker .

EXPOSE 8080

CMD ["./server"]
