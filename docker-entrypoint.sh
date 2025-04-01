#!/bin/bash
set -e

echo "欢迎使用Amazon爬虫工具"
echo "正在启动静态文件服务..."

# 检查是否已安装nginx
if ! command -v nginx &> /dev/null; then
    echo "正在安装nginx..."
    apt-get update
    apt-get install -y nginx
    apt-get clean
    rm -rf /var/lib/apt/lists/*
fi

# 创建nginx配置
cat > /etc/nginx/sites-available/default << EOF
server {
    listen 8899 default_server;
    listen [::]:8899 default_server;

    root /app;
    index index.html;

    server_name _;

    location / {
        try_files \$uri \$uri/ /templates/index.html;
    }

    location /static/ {
        alias /app/static/;
    }

    location /api/ {
        return 200 '{"status":"success","message":"API暂不可用，正在维护中"}';
        add_header Content-Type application/json;
    }
}
EOF

# 启动nginx
echo "启动服务..."
nginx -g "daemon off;" 