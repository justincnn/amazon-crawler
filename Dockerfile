FROM golang:1.19 AS builder

WORKDIR /app

# 确保所有依赖已安装
RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev gcc

# 复制所有文件
COPY . .

# 列出错误模块
RUN go mod tidy

# 禁用CGO的编译尝试
ENV CGO_ENABLED=0
RUN go build -tags nocgo -o amazon-crawler-nocgo .

# 如果没有CGO不工作，我们尝试启用CGO编译
ENV CGO_ENABLED=1
RUN go build -o amazon-crawler .

FROM debian:bullseye-slim

WORKDIR /app

# 安装运行时依赖
RUN apt-get update && apt-get install -y ca-certificates tzdata sqlite3 && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# 尝试复制正确的编译结果
COPY --from=builder /app/amazon-crawler* /app/amazon-crawler
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