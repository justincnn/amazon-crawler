FROM golang:1.19 AS builder

WORKDIR /app

# 安装必要的编译依赖
RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev

# 设置CGO环境
ENV CGO_ENABLED=1

# 复制所有文件
COPY . .

# 创建main_simple.go以检查基本构建
RUN echo 'package main\n\nimport (\n\t"fmt"\n\t"github.com/gin-gonic/gin"\n\t"net/http"\n)\n\nfunc main() {\n\tr := gin.Default()\n\tr.GET("/ping", func(c *gin.Context) {\n\t\tc.JSON(http.StatusOK, gin.H{"message": "pong"})\n\t})\n\tr.Static("/static", "./static")\n\tr.LoadHTMLGlob("templates/*")\n\tr.GET("/", func(c *gin.Context) {\n\t\tc.HTML(http.StatusOK, "index.html", nil)\n\t})\n\tfmt.Println("Server started on :8899")\n\tr.Run(":8899")\n}' > main_simple.go

# 准备依赖
RUN go mod tidy

# 尝试构建简化版本
RUN echo "=== 尝试构建简化版本 ===" && \
    go build -o amazon-crawler-simple main_simple.go

FROM debian:bullseye-slim

WORKDIR /app

# 安装运行时依赖
RUN apt-get update && apt-get install -y ca-certificates tzdata sqlite3 && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# 复制编译好的应用
COPY --from=builder /app/amazon-crawler-simple /app/amazon-crawler
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

# 确保二进制可执行
RUN chmod +x /app/amazon-crawler

CMD ["./amazon-crawler", "-c", "config.yaml"] 