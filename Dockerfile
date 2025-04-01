FROM golang:1.19 AS builder

WORKDIR /app

# 显示Go环境信息
RUN go version && go env

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./
# 下载依赖（显示详细日志）
RUN go mod download -x

# 复制源代码
COPY . .

# 先检查代码问题
RUN go vet ./...

# 编译应用（添加详细调试信息）
RUN go build -v -x -o amazon-crawler . 2>&1 || \
    (echo "===== 构建失败! 错误详情: =====" && \
     go build -v -x -o amazon-crawler . 2>&1 && \
     echo "================================" && \
     exit 1)

FROM debian:bullseye-slim

WORKDIR /app

# 安装运行时依赖
RUN apt-get update && apt-get install -y ca-certificates tzdata sqlite3 && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# 复制编译好的应用
COPY --from=builder /app/amazon-crawler /app/
COPY --from=builder /app/config.yaml.save /app/config.yaml
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/static /app/static

# 创建数据目录
RUN mkdir -p /app/data
RUN mkdir -p /app/logs

# 设置环境变量
ENV GIN_MODE=release

# 暴露Web端口
EXPOSE 8899

# 设置数据卷
VOLUME ["/app/data", "/app/logs"]

# 设置启动命令
CMD ["./amazon-crawler", "-c", "config.yaml"] 