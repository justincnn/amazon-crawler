package main

import (
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/tengfei-xy/go-log"
	"gopkg.in/yaml.v3"
)

const SQLITE_APPLICATION_STATUS_START int = 0
const SQLITE_APPLICATION_STATUS_OVER int = 1
const SQLITE_APPLICATION_STATUS_SEARCH int = 2
const SQLITE_APPLICATION_STATUS_PRODUCT int = 3
const SQLITE_APPLICATION_STATUS_SELLER int = 4

type appConfig struct {
	Basic      `yaml:"basic"`
	Proxy      `yaml:"proxy"`
	Exec       `yaml:"exec"`
	db         *sql.DB
	cookie     string
	primary_id int64
	logFile    *os.File
	logMutex   sync.Mutex
}
type Exec struct {
	Enable          `yaml:"enable"`
	Loop            `yaml:"loop"`
	Search_priority int `yaml:"search_priority"`
}
type Enable struct {
	Search  bool `yaml:"search"`
	Product bool `yaml:"product"`
	Seller  bool `yaml:"seller"`
}
type Loop struct {
	All          int `yaml:"all"`
	all_time     int
	Search       int `yaml:"search"`
	search_time  int
	Product      int `yaml:"product"`
	product_time int
	Seller       int `yaml:"seller"`
	seller_time  int
}
type Basic struct {
	App_id     int    `yaml:"app_id"`
	Host_id    int    `yaml:"host_id"`
	Test       bool   `yaml:"test"`
	Domain     string `yaml:"domain"`
	DbPath     string `yaml:"db_path"`
	ServerPort string `yaml:"server_port"`
	LogPath    string `yaml:"log_path"`
}
type Proxy struct {
	Enable bool `yaml:"enable"`
	Sockc5 []string `yaml:"socks5"`
}
type flagStruct struct {
	config_file string
}

// 全局变量
var app appConfig
var robot Robots
var templates *template.Template

const userAgent = `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36`

// 初始化日志文件
func initLogFile() error {
	// 设置默认日志路径
	if app.Basic.LogPath == "" {
		app.Basic.LogPath = "logs"
	}
	
	// 确保日志目录存在
	if err := os.MkdirAll(app.Basic.LogPath, 0755); err != nil {
		return err
	}
	
	// 创建按日期命名的日志文件
	logFileName := filepath.Join(app.Basic.LogPath, time.Now().Format("2006-01-02")+".log")
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	
	// 设置日志输出同时到文件和控制台
	log.SetOutput(io.MultiWriter(os.Stdout, file))
	
	// 保存文件句柄以便后续使用
	app.logFile = file
	
	return nil
}

func init_config(flag flagStruct) {
	log.Infof("读取配置文件:%s", flag.config_file)

	yamlFile, err := os.ReadFile(flag.config_file)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlFile, &app)
	if err != nil {
		panic(err)
	}
	
	// 默认值设置
	if app.Basic.ServerPort == "" {
		app.Basic.ServerPort = "8080"
	}
	if app.Basic.DbPath == "" {
		app.Basic.DbPath = "amazon.db"
	}
	if !app.Exec.Enable.Search && !app.Exec.Enable.Product && !app.Exec.Enable.Seller {
		panic("没有启动功能，检查配置文件的enable配置的选项")
	}
	if app.Exec.Loop.All == 0 {
		app.Exec.Loop.All = 999999
	}
	if app.Exec.Loop.Search == 0 {
		app.Exec.Loop.Search = 999999
	}
	if app.Exec.Loop.Product == 0 {
		app.Exec.Loop.Product = 999999
	}
	if app.Exec.Loop.Seller == 0 {
		app.Exec.Loop.Seller = 999999
	}
	app.Exec.product_time = 0
	app.Exec.search_time = 0
	app.Exec.seller_time = 0

	log.Infof("程序标识:%d 主机标识:%d", app.Basic.App_id, app.Basic.Host_id)
	
	// 初始化日志文件
	if err := initLogFile(); err != nil {
		log.Errorf("初始化日志文件失败: %v", err)
	}
}

func init_rebots() {
	robotTxt := fmt.Sprintf("https://%s/robots.txt", app.Domain)

	log.Infof("加载文件: %s", robotTxt)
	txt, err := request_get(robotTxt, userAgent)
	if err != nil {
		log.Error("网络错误")
		panic(err)
	}
	robot = GetRobotFromTxt(txt)
}

func init_sqlite() {
	// 检查数据库文件是否存在
	_, err := os.Stat(app.Basic.DbPath)
	dbExists := !os.IsNotExist(err)
	
	// 连接数据库
	db, err := sql.Open("sqlite3", app.Basic.DbPath)
	if err != nil {
		panic(err)
	}
	
	if err := db.Ping(); err != nil {
		panic(err)
	}
	
	// 如果数据库文件不存在，创建表结构
	if !dbExists {
		log.Info("初始化数据库表结构")
		if err := createDatabaseSchema(db); err != nil {
			panic(err)
		}
	}
	
	log.Info("数据库已连接")
	app.db = db
}

func createDatabaseSchema(db *sql.DB) error {
	// 创建application表
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS application (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		app_id INTEGER NOT NULL,
		status INTEGER NOT NULL DEFAULT 0,
		update_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}
	
	// 创建category表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS category (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		zh_key TEXT NOT NULL,
		en_key TEXT NOT NULL,
		priority INTEGER DEFAULT 0
	)`)
	if err != nil {
		return err
	}
	
	// 创建cookie表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS cookie (
		host_id INTEGER PRIMARY KEY,
		cookie TEXT
	)`)
	if err != nil {
		return err
	}
	
	// 创建product表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS product (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL UNIQUE,
		param TEXT NOT NULL,
		status INTEGER DEFAULT 0,
		app INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return err
	}
	
	// 创建search_statistics表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS search_statistics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category_id INTEGER NOT NULL,
		start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		end TIMESTAMP,
		status INTEGER DEFAULT 0,
		app INTEGER NOT NULL,
		valid INTEGER DEFAULT 0
	)`)
	if err != nil {
		return err
	}
	
	// 创建seller表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS seller (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		seller_id TEXT UNIQUE,
		status INTEGER DEFAULT 0,
		app INTEGER NOT NULL DEFAULT 0,
		name TEXT,
		address TEXT,
		trn TEXT,
		info_flag INTEGER DEFAULT 0,
		trn_flag INTEGER DEFAULT 0,
		inserted TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	
	return err
}

func init_network() {
	log.Info("网络测试开始")

	var s searchStruct
	s.en_key = "Hardware+electrician"
	_, err := s.request(0)
	if err != nil {
		log.Error("网络错误")
		panic(err)
	}
}

func init_signal() {
	// 创建一个通道来接收操作系统的信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGABRT)

	go func() {
		<-sigCh
		log.Info("")
		log.Infof("程序即将结束")
		app.end()
		app.db.Close()
		if app.logFile != nil {
			app.logFile.Close()
		}
		log.Infof("程序结束")
		os.Exit(0)
	}()
}

func init_flag() flagStruct {
	var f flagStruct
	flag.StringVar(&f.config_file, "c", "config.yaml", "打开配置文件")
	flag.Parse()
	return f
}

func setupRouter() *gin.Engine {
	// 设置gin为release模式
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	
	r := gin.Default()
	
	// 设置CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
		AllowCredentials: true,
	}))
	
	// 加载静态文件
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")
	
	// 首页
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "亚马逊爬虫工具",
		})
	})
	
	// API路由
	api := r.Group("/api")
	{
		// 获取爬虫状态
		api.GET("/status", getStatus)
		
		// 关键词管理
		api.GET("/keywords", getKeywords)
		api.POST("/keywords", addKeyword)
		api.DELETE("/keywords/:id", deleteKeyword)
		
		// 启动爬虫
		api.POST("/crawler/start", startCrawler)
		
		// 获取爬虫结果
		api.GET("/results", getResults)
		api.GET("/sellers", getSellers)
		
		// 获取配置
		api.GET("/config", getConfig)
		api.POST("/config", updateConfig)
		
		// 获取cookie
		api.GET("/cookie", getCookie)
		api.POST("/cookie", updateCookie)
		
		// 获取日志
		api.GET("/logs", getLogs)
	}
	
	return r
}

func main() {
	f := init_flag()
	init_config(f)
	init_rebots()
	init_sqlite()
	init_network()
	init_signal()

	app.start()
	
	// 启动爬虫后台任务
	go runCrawlerTasks()
	
	// 启动Web服务器
	r := setupRouter()
	log.Infof("Web服务器已启动，端口: %s", app.Basic.ServerPort)
	r.Run(":" + app.Basic.ServerPort)
}

// 后台运行爬虫任务
func runCrawlerTasks() {
	for app.Exec.Loop.all_time = 0; app.Exec.Loop.all_time < app.Exec.Loop.All; app.Exec.Loop.all_time++ {
		if app.Exec.Enable.Search {
			var search searchStruct
			search.main()
		}

		if app.Exec.Enable.Product {
			var product productStruct
			product.main()
		}

		if app.Exec.Enable.Seller {
			var seller sellerStruct
			seller.main()
		}
		
		// 休息一段时间，避免CPU占用过高
		time.Sleep(5 * time.Second)
	}
}

// API处理函数
func getStatus(c *gin.Context) {
	status := map[string]interface{}{
		"search_enabled":  app.Exec.Enable.Search,
		"product_enabled": app.Exec.Enable.Product,
		"seller_enabled":  app.Exec.Enable.Seller,
		"search_times":    app.Exec.Loop.search_time,
		"product_times":   app.Exec.Loop.product_time,
		"seller_times":    app.Exec.Loop.seller_time,
	}
	
	c.JSON(http.StatusOK, status)
}

func getKeywords(c *gin.Context) {
	rows, err := app.db.Query("SELECT id, zh_key, en_key, priority FROM category ORDER BY priority DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var keywords []map[string]interface{}
	for rows.Next() {
		var id int
		var zhKey, enKey string
		var priority int
		if err := rows.Scan(&id, &zhKey, &enKey, &priority); err != nil {
			continue
		}
		
		keywords = append(keywords, map[string]interface{}{
			"id":       id,
			"zh_key":   zhKey,
			"en_key":   enKey,
			"priority": priority,
		})
	}
	
	c.JSON(http.StatusOK, keywords)
}

func addKeyword(c *gin.Context) {
	var keyword struct {
		ZhKey    string `json:"zh_key" binding:"required"`
		EnKey    string `json:"en_key" binding:"required"`
		Priority int    `json:"priority"`
	}
	
	if err := c.BindJSON(&keyword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 验证输入
	if keyword.ZhKey == "" || keyword.EnKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "中文关键词和英文关键词都不能为空"})
		return
	}
	
	res, err := app.db.Exec("INSERT INTO category (zh_key, en_key, priority) VALUES (?, ?, ?)",
		keyword.ZhKey, keyword.EnKey, keyword.Priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	id, _ := res.LastInsertId()
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "success"})
}

func deleteKeyword(c *gin.Context) {
	id := c.Param("id")
	_, err := app.db.Exec("DELETE FROM category WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func startCrawler(c *gin.Context) {
	var config struct {
		Search      bool `json:"search"`
		Product     bool `json:"product"`
		Seller      bool `json:"seller"`
		LoopAll     int  `json:"loop_all"`
		LoopSearch  int  `json:"loop_search"`
		LoopProduct int  `json:"loop_product"`
		LoopSeller  int  `json:"loop_seller"`
	}
	
	if err := c.BindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	app.Exec.Enable.Search = config.Search
	app.Exec.Enable.Product = config.Product
	app.Exec.Enable.Seller = config.Seller
	
	// 设置循环次数
	if config.LoopAll >= 0 {
		app.Exec.Loop.All = config.LoopAll
	}
	if config.LoopSearch >= 0 {
		app.Exec.Loop.Search = config.LoopSearch
	}
	if config.LoopProduct >= 0 {
		app.Exec.Loop.Product = config.LoopProduct
	}
	if config.LoopSeller >= 0 {
		app.Exec.Loop.Seller = config.LoopSeller
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

func getResults(c *gin.Context) {
	limit := c.DefaultQuery("limit", "100")
	offset := c.DefaultQuery("offset", "0")
	
	rows, err := app.db.Query(`
		SELECT p.id, p.url, p.param, p.status, c.zh_key, c.en_key 
		FROM product p
		LEFT JOIN search_statistics s ON p.id = s.id
		LEFT JOIN category c ON s.category_id = c.id
		ORDER BY p.id DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var products []map[string]interface{}
	for rows.Next() {
		var id int
		var url, param string
		var status int
		var zhKey, enKey sql.NullString
		
		if err := rows.Scan(&id, &url, &param, &status, &zhKey, &enKey); err != nil {
			continue
		}
		
		products = append(products, map[string]interface{}{
			"id":     id,
			"url":    url,
			"param":  param,
			"status": status,
			"zh_key": zhKey.String,
			"en_key": enKey.String,
		})
	}
	
	c.JSON(http.StatusOK, products)
}

func getSellers(c *gin.Context) {
	limit := c.DefaultQuery("limit", "100")
	offset := c.DefaultQuery("offset", "0")
	query := c.DefaultQuery("query", "")
	
	sqlQuery := `
		SELECT id, seller_id, status, name, address, trn, info_flag, trn_flag
		FROM seller
		WHERE 1=1
	`
	var args []interface{}
	
	if query != "" {
		sqlQuery += " AND (seller_id LIKE ? OR name LIKE ? OR address LIKE ? OR trn LIKE ?)"
		queryParam := "%" + query + "%"
		args = append(args, queryParam, queryParam, queryParam, queryParam)
	}
	
	sqlQuery += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	
	rows, err := app.db.Query(sqlQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var sellers []map[string]interface{}
	for rows.Next() {
		var id int
		var sellerID string
		var status, infoFlag, trnFlag int
		var name, address, trn sql.NullString
		
		if err := rows.Scan(&id, &sellerID, &status, &name, &address, &trn, &infoFlag, &trnFlag); err != nil {
			continue
		}
		
		sellers = append(sellers, map[string]interface{}{
			"id":        id,
			"seller_id": sellerID,
			"status":    status,
			"name":      name.String,
			"address":   address.String,
			"trn":       trn.String,
			"info_flag": infoFlag,
			"trn_flag":  trnFlag,
		})
	}
	
	c.JSON(http.StatusOK, sellers)
}

func getConfig(c *gin.Context) {
	config := map[string]interface{}{
		"app_id":           app.Basic.App_id,
		"host_id":          app.Basic.Host_id,
		"domain":           app.Basic.Domain,
		"search_enabled":   app.Exec.Enable.Search,
		"product_enabled":  app.Exec.Enable.Product,
		"seller_enabled":   app.Exec.Enable.Seller,
		"search_priority":  app.Exec.Search_priority,
		"proxy_enabled":    app.Proxy.Enable,
		"proxy_socks5":     app.Proxy.Sockc5,
		"loop_all":         app.Exec.Loop.All,
		"loop_search":      app.Exec.Loop.Search,
		"loop_product":     app.Exec.Loop.Product,
		"loop_seller":      app.Exec.Loop.Seller,
	}
	
	c.JSON(http.StatusOK, config)
}

func updateConfig(c *gin.Context) {
	var config struct {
		AppID          int      `json:"app_id"`
		HostID         int      `json:"host_id"`
		Domain         string   `json:"domain"`
		SearchEnabled  bool     `json:"search_enabled"`
		ProductEnabled bool     `json:"product_enabled"`
		SellerEnabled  bool     `json:"seller_enabled"`
		SearchPriority int      `json:"search_priority"`
		ProxyEnabled   bool     `json:"proxy_enabled"`
		ProxySocks5    []string `json:"proxy_socks5"`
	}
	
	if err := c.BindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	app.Basic.App_id = config.AppID
	app.Basic.Host_id = config.HostID
	app.Basic.Domain = config.Domain
	app.Exec.Enable.Search = config.SearchEnabled
	app.Exec.Enable.Product = config.ProductEnabled
	app.Exec.Enable.Seller = config.SellerEnabled
	app.Exec.Search_priority = config.SearchPriority
	app.Proxy.Enable = config.ProxyEnabled
	app.Proxy.Sockc5 = config.ProxySocks5
	
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func getCookie(c *gin.Context) {
	var cookie string
	err := app.db.QueryRow("SELECT cookie FROM cookie WHERE host_id = ?", app.Basic.Host_id).Scan(&cookie)
	if err != nil {
		cookie = ""
	}
	
	c.JSON(http.StatusOK, gin.H{"cookie": cookie})
}

func updateCookie(c *gin.Context) {
	var cookieData struct {
		Cookie string `json:"cookie" binding:"required"`
	}
	
	if err := c.BindJSON(&cookieData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 尝试更新
	res, err := app.db.Exec("UPDATE cookie SET cookie = ? WHERE host_id = ?", 
		cookieData.Cookie, app.Basic.Host_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		// 没有记录，插入新记录
		_, err = app.db.Exec("INSERT INTO cookie (host_id, cookie) VALUES (?, ?)",
			app.Basic.Host_id, cookieData.Cookie)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	
	app.cookie = cookieData.Cookie
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// 获取日志
func getLogs(c *gin.Context) {
	level := c.DefaultQuery("level", "all")
	linesStr := c.DefaultQuery("lines", "100")
	
	lines, err := strconv.Atoi(linesStr)
	if err != nil || lines <= 0 {
		lines = 100
	}
	if lines > 1000 {
		lines = 1000
	}
	
	// 确定日志文件路径
	logFileName := filepath.Join(app.Basic.LogPath, time.Now().Format("2006-01-02")+".log")
	
	app.logMutex.Lock()
	defer app.logMutex.Unlock()
	
	// 读取日志文件
	logData, err := os.ReadFile(logFileName)
	if err != nil {
		// 如果今天的日志文件不存在，尝试找最近的日志文件
		files, err := os.ReadDir(app.Basic.LogPath)
		if err != nil || len(files) == 0 {
			c.JSON(http.StatusOK, gin.H{"logs": []string{}})
			return
		}
		
		// 按名称排序（日期格式排序）
		var logFiles []string
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".log") {
				logFiles = append(logFiles, file.Name())
			}
		}
		
		if len(logFiles) == 0 {
			c.JSON(http.StatusOK, gin.H{"logs": []string{}})
			return
		}
		
		// 读取最新的日志文件
		logFileName = filepath.Join(app.Basic.LogPath, logFiles[len(logFiles)-1])
		logData, err = os.ReadFile(logFileName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取日志文件"})
			return
		}
	}
	
	// 按行分割
	allLines := strings.Split(string(logData), "\n")
	
	// 根据日志级别过滤
	var filteredLines []string
	switch level {
	case "info":
		filteredLines = filterLogsByPattern(allLines, "\\[INFO\\]")
	case "warn":
		filteredLines = filterLogsByPattern(allLines, "\\[WARN\\]")
	case "error":
		filteredLines = filterLogsByPattern(allLines, "\\[ERROR\\]")
	default:
		filteredLines = allLines
	}
	
	// 取最后n行
	resultLines := getLastNLines(filteredLines, lines)
	
	c.JSON(http.StatusOK, gin.H{"logs": resultLines})
}

// 按正则表达式过滤日志行
func filterLogsByPattern(lines []string, pattern string) []string {
	re := regexp.MustCompile(pattern)
	var filtered []string
	
	for _, line := range lines {
		if re.MatchString(line) {
			filtered = append(filtered, line)
		}
	}
	
	return filtered
}

// 获取最后n行
func getLastNLines(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

func (app *appConfig) get_cookie() (string, error) {
	var cookie string
	if app.Basic.Host_id == 0 {
		return "", fmt.Errorf("配置文件中host_id为0，cookie将为空")
	}

	err := app.db.QueryRow("SELECT cookie FROM cookie WHERE host_id = ?", app.Basic.Host_id).Scan(&cookie)
	if err != nil {
		return "", err
	}
	cookie = strings.TrimSpace(cookie)
	if app.cookie != cookie {
		log.Infof("使用新cookie: %s", cookie)
	}

	app.cookie = cookie
	return app.cookie, nil
}

func (app *appConfig) start() {
	if app.Basic.Test {
		log.Infof("测试模式启动")
		return
	}
	r, err := app.db.Exec("INSERT INTO application (app_id) VALUES(?)", app.Basic.App_id)
	if err != nil {
		panic(err)
	}
	id, err := r.LastInsertId()
	if err != nil {
		panic(err)
	}
	app.primary_id = id
}

func (app *appConfig) update(status int) {
	_, err := app.db.Exec("UPDATE application SET status=?, update_time=CURRENT_TIMESTAMP WHERE id=?", status, app.primary_id)
	if err != nil {
		panic(err)
	}
}

func (app *appConfig) end() {
	if app.Basic.Test {
		return
	}
	if _, err := app.db.Exec("UPDATE application SET status=?, update_time=CURRENT_TIMESTAMP WHERE id=?", SQLITE_APPLICATION_STATUS_OVER, app.primary_id); err != nil {
		log.Error(err)
	}
}
