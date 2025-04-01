FROM golang:1.19 AS builder

WORKDIR /app

# 安装必要的编译依赖
RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev gcc pkg-config

# 显示环境变量
RUN env

# 设置CGO环境
ENV CGO_ENABLED=1 
ENV PKG_CONFIG_PATH=/usr/lib/pkgconfig

# 复制所有文件
COPY . .

# 准备依赖
RUN go mod tidy
RUN go mod download -x

# 尝试单独编译sqlite驱动测试CGO
RUN go build -v -x github.com/mattn/go-sqlite3 

# 编译应用（添加详细输出）
RUN go build -v -o amazon-crawler . || (go build -v -x -o amazon-crawler . 2>&1 && exit 1)

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