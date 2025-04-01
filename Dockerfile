FROM debian:bullseye-slim

WORKDIR /app

# 安装基本依赖
RUN apt-get update && apt-get install -y ca-certificates tzdata sqlite3 && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# 创建配置文件
COPY config.yaml.save /app/config.yaml

# 复制静态资源和模板
COPY templates /app/templates
COPY static /app/static

# 创建数据目录和日志目录
RUN mkdir -p /app/data
RUN mkdir -p /app/logs

# 设置容器入口
COPY docker-entrypoint.sh /app/
RUN chmod +x /app/docker-entrypoint.sh

# 暴露Web端口
EXPOSE 8899

# 设置数据卷
VOLUME ["/app/data", "/app/logs"]

# 设置启动命令
ENTRYPOINT ["/app/docker-entrypoint.sh"] 