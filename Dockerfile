FROM golang:1.19 AS builder

WORKDIR /app

# 安装必要的编译依赖
RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev gcc pkg-config

# 设置CGO环境
ENV CGO_ENABLED=1 

# 复制所有文件
COPY . .

# 列出所有go文件
RUN find . -name "*.go" -type f | sort

# 准备依赖
RUN go mod tidy

# 尝试编译不同模块以找出问题
RUN echo "=== 尝试编译不同模块 ===" && \
    go build -v github.com/mattn/go-sqlite3 && \
    echo "Go-SQLite3 编译成功!"

# 使用纯静态编译尝试
RUN echo "=== 尝试纯静态编译 ===" && \
    CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o amazon-crawler-static . || echo "静态编译失败，尝试普通编译"

# 使用简单编译
RUN echo "=== 尝试普通编译 ===" && \
    CGO_ENABLED=1 go build -o amazon-crawler .

FROM debian:bullseye-slim

WORKDIR /app

# 安装运行时依赖
RUN apt-get update && apt-get install -y ca-certificates tzdata sqlite3 && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# 复制编译好的应用 - 尝试两种可能的二进制名称
COPY --from=builder /app/amazon-crawler* /app/
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

# 设置启动命令（处理不同的可能二进制名称）
RUN if [ -f "/app/amazon-crawler" ]; then chmod +x /app/amazon-crawler; \
    elif [ -f "/app/amazon-crawler-static" ]; then cp /app/amazon-crawler-static /app/amazon-crawler && chmod +x /app/amazon-crawler; \
    else echo "没有找到有效的二进制文件"; exit 1; fi

CMD ["./amazon-crawler", "-c", "config.yaml"] 