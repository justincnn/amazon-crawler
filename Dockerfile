FROM golang:1.19-alpine AS builder

WORKDIR /app

# 安装编译依赖
RUN apk add --no-cache gcc musl-dev git

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译应用(修改编译方式)
RUN go build -o amazon-crawler .

# 使用alpine作为基础镜像，减小镜像大小
FROM alpine:latest

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata sqlite && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 复制编译好的应用
COPY --from=builder /app/amazon-crawler /app/
COPY --from=builder /app/config.yaml.save /app/config.yaml
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/static /app/static

# 创建数据目录
RUN mkdir -p /app/data

# 设置环境变量
ENV GIN_MODE=release

# 暴露Web端口
EXPOSE 8899

# 设置数据卷
VOLUME ["/app/data"]

# 设置启动命令
CMD ["./amazon-crawler", "-c", "config.yaml"] 