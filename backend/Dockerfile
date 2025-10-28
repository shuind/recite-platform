# --- 阶段 1: 构建阶段 ---
# 使用官方的 Go 镜像作为构建环境，并给这个阶段命名为 "builder"
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件，并下载依赖
# 这一步可以利用 Docker 的缓存机制，如果依赖没变，就不用重新下载
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码到工作目录
COPY . .

# 编译 Go 应用。
# CGO_ENABLED=0: 关闭 CGO，为了生成静态链接的可执行文件。
# GOOS=linux: 指定编译为 Linux 系统下的可执行文件，因为我们的最终容器是 Alpine Linux。
# -o /server: 指定输出的可执行文件名为 server，并放在根目录。
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker/main.go

# --- 阶段 2: 运行阶段 ---
# 使用一个非常小的基础镜像 (alpine)，减少最终镜像的大小
FROM alpine:latest

# 设置工作目录
WORKDIR /root/

# 从 "builder" 阶段复制编译好的可执行文件到当前镜像
COPY --from=builder /server .
COPY --from=builder /worker .
# (可选) 如果有配置文件，也从 builder 阶段复制过来
# COPY --from=builder /app/config.yml .

# 暴露端口 8080，让容器外的服务可以访问
EXPOSE 8080

# 容器启动时要执行的命令：运行我们的 Go 程序
CMD ["./server"]