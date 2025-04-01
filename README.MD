# amazon-crawler

一种在亚马逊平台，根据关键词支持多主机搜索商家信息（名称、地址、税号）的工具，适用多种主流操作系统。带有Web界面，可直接通过浏览器进行操作。


![Docker自动构建](https://github.com/用户名/amazon-crawler/actions/workflows/docker-build.yml/badge.svg)

# 一、爬虫说明

请尊守网站的robots.txt，本项目中的程序的任何http请求均取决于robots.txt

# 二、项目说明

本项目是通过关键词在亚马逊进行搜索的，并将搜索结果保留到数据库中并提取每个商品的卖家的信息，最后从卖家信息中提取TRN。由于每个步骤相互独立，因此适合分布式运行。

## 新功能

- Web界面：添加了Web前端界面，可以直接通过浏览器操作爬虫
- SQLite数据库：使用SQLite替代MySQL，降低部署复杂度
- 关键词管理：通过Web界面添加和管理关键词
- 爬虫结果显示：在Web界面上直接查看和筛选爬虫结果
- Docker部署：支持通过Docker快速部署
- CI/CD：GitHub Actions自动构建Docker镜像并推送到Docker Hub

# 三、使用说明

## 方法一：二进制运行

1. [下载](https://github.com/tengfei-xy/amazon-crawler/releases)最新版本

2. 创建并修改配置文件

   ```bash
   cp config.yaml.save config.yaml
   vim config.yaml  # 修改配置
   ```

3. 启动程序

   ```bash
   ./amazon-crawler -c config.yaml
   ```

4. 打开浏览器，访问 http://localhost:8080 (或配置的其他端口)

## 方法二：Docker部署（本地构建）

1. 克隆仓库

   ```bash
   git clone https://github.com/用户名/amazon-crawler.git
   cd amazon-crawler
   ```

2. 使用Docker Compose构建并启动

   ```bash
   docker-compose up -d
   ```

3. 打开浏览器，访问 http://localhost:8080

## 方法三：直接从Docker Hub拉取镜像

```bash
docker pull 用户名/amazon-crawler:latest
docker run -d -p 8080:8080 -v ./data:/app/data 用户名/amazon-crawler:latest
```

# 四、Web界面使用

## 控制面板

控制面板用于控制爬虫的启动和监控爬虫状态：

- 启用/禁用各种爬虫功能
- 查看爬虫运行统计信息
- 启动爬虫

## 关键词管理

在关键词管理页面可以：

- 查看已有关键词列表
- 添加新的关键词（中英文）
- 设置关键词优先级
- 删除不需要的关键词

## 产品列表

产品列表页面显示爬虫发现的所有产品：

- 查看产品URL
- 查看产品状态
- 查看关联的关键词

## 商家信息

商家信息页面显示从产品中提取的商家数据：

- 查看商家名称、地址
- 查看税号信息
- 按税号搜索筛选商家

## 设置

设置页面用于配置爬虫：

- 设置程序标识和主机标识
- 设置亚马逊域名
- 配置Cookie信息
- 设置搜索优先级规则

# 五、数据库结构

项目使用SQLite数据库，主要表结构如下：

- application：记录程序启动状态
- category：存储搜索关键词
- cookie：存储不同主机的cookie
- product：存储搜索到的产品URL
- search_statistics：记录搜索统计信息
- seller：存储商家信息

# 六、注意事项

- Cookie问题：亚马逊的cookie有效期较短，需要定期更新
- 网络环境：建议使用代理，避免IP被封
- 爬虫频率：请合理控制爬取频率，避免对目标网站造成压力

# 七、CI/CD自动构建

本项目使用GitHub Actions自动构建Docker镜像并推送到Docker Hub。当推送代码到main或master分支，或创建新的标签（格式为v*）时，将自动触发构建流程。

## 设置GitHub Secrets

要启用自动构建，需要在GitHub仓库中设置以下Secrets：

1. `DOCKERHUB_USERNAME` - 你的Docker Hub用户名
2. `DOCKERHUB_TOKEN` - 你的Docker Hub访问令牌（不是密码）

### 获取Docker Hub访问令牌

1. 登录到[Docker Hub](https://hub.docker.com/)
2. 点击右上角你的头像，选择"Account Settings"
3. 在左侧菜单中选择"Security"
4. 点击"New Access Token"
5. 输入令牌名称（例如"GitHub Actions"）并选择权限
6. 点击"Generate"
7. 复制生成的令牌（只显示一次）并添加到GitHub Secrets 
