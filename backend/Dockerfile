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
RUN apk add --no-cache tzdata
ENV TZ=Asia/Shanghai
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezone
WORKDIR /root/
COPY --from=builder /server .
COPY --from=builder /worker .
EXPOSE 8080
CMD ["./server"]