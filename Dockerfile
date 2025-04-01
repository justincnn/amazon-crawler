FROM golang:1.19 AS builder

WORKDIR /app

# 显示Go环境信息
RUN go version && go env

# 复制所有文件
COPY . .

# 安装sqlite3驱动依赖
RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev

# 确保依赖已下载
RUN go mod download

# 编译应用（简化为基本命令）
RUN go build -o amazon-crawler .

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