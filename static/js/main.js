// 页面加载完成后执行
document.addEventListener('DOMContentLoaded', function() {
    // 导航切换
    const navLinks = document.querySelectorAll('.nav-link');
    navLinks.forEach(link => {
        link.addEventListener('click', function() {
            // 移除所有当前激活的nav-link和content-section
            document.querySelectorAll('.nav-link.active').forEach(el => el.classList.remove('active'));
            document.querySelectorAll('.content-section.active').forEach(el => el.classList.remove('active'));
            
            // 添加当前激活的nav-link
            this.classList.add('active');
            
            // 显示对应的content-section
            const targetId = this.getAttribute('data-bs-target');
            document.getElementById(targetId).classList.add('active');
            
            // 如果切换到日志页面，自动加载日志
            if (targetId === 'logs-section') {
                loadLogs();
            }
        });
    });

    // 初始化数据
    loadStatus();
    loadKeywords();
    loadProducts(0);
    loadSellers(0);
    loadConfig();
    loadCookie();

    // 启动爬虫按钮点击事件
    document.getElementById('startCrawlerBtn').addEventListener('click', startCrawler);
    
    // 停止爬虫按钮点击事件
    document.getElementById('stopCrawlerBtn').addEventListener('click', stopCrawler);

    // 保存关键词按钮点击事件
    document.getElementById('saveKeywordBtn').addEventListener('click', saveKeyword);

    // 搜索税号按钮点击事件
    document.getElementById('searchTrnBtn').addEventListener('click', function() {
        loadSellers(0, document.getElementById('trnSearch').value);
    });

    // 配置表单提交事件
    document.getElementById('configForm').addEventListener('submit', function(e) {
        e.preventDefault();
        saveConfig();
    });

    // Cookie表单提交事件
    document.getElementById('cookieForm').addEventListener('submit', function(e) {
        e.preventDefault();
        saveCookie();
    });
    
    // 日志刷新按钮点击事件
    document.getElementById('refreshLogBtn').addEventListener('click', loadLogs);
    
    // 日志筛选事件
    document.getElementById('logLevelSelect').addEventListener('change', loadLogs);
    
    // 日志行数变更事件
    document.getElementById('logLines').addEventListener('change', loadLogs);

    // 定期更新状态
    setInterval(loadStatus, 5000);
});

// 加载爬虫状态
function loadStatus() {
    fetch('/api/status')
        .then(response => response.json())
        .then(data => {
            document.getElementById('searchToggle').checked = data.search_enabled;
            document.getElementById('productToggle').checked = data.product_enabled;
            document.getElementById('sellerToggle').checked = data.seller_enabled;
            document.getElementById('searchTimes').textContent = data.search_times;
            document.getElementById('productTimes').textContent = data.product_times;
            document.getElementById('sellerTimes').textContent = data.seller_times;
            
            // 更新UI，根据爬虫状态
            const startBtn = document.getElementById('startCrawlerBtn');
            const stopBtn = document.getElementById('stopCrawlerBtn');
            
            switch(data.crawler_status) {
                case 'running':
                    startBtn.disabled = true;
                    stopBtn.disabled = false;
                    startBtn.textContent = '爬虫运行中';
                    break;
                case 'stopping':
                    startBtn.disabled = true;
                    stopBtn.disabled = true;
                    startBtn.textContent = '爬虫停止中';
                    break;
                case 'stopped':
                default:
                    startBtn.disabled = false;
                    stopBtn.disabled = true;
                    startBtn.textContent = '启动爬虫';
                    break;
            }
        })
        .catch(error => console.error('Error:', error));
}

// 启动爬虫
function startCrawler() {
    const config = {
        search: document.getElementById('searchToggle').checked,
        product: document.getElementById('productToggle').checked,
        seller: document.getElementById('sellerToggle').checked,
        loop_all: parseInt(document.getElementById('allLoopCount').value) || 0,
        loop_search: parseInt(document.getElementById('searchLoopCount').value) || 0,
        loop_product: parseInt(document.getElementById('productLoopCount').value) || 0,
        loop_seller: parseInt(document.getElementById('sellerLoopCount').value) || 0
    };

    fetch('/api/crawler/start', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(config)
    })
    .then(response => response.json())
    .then(data => {
        alert('爬虫已启动！');
        loadStatus();
    })
    .catch(error => console.error('Error:', error));
}

// 停止爬虫
function stopCrawler() {
    fetch('/api/crawler/stop', {
        method: 'POST'
    })
    .then(response => response.json())
    .then(data => {
        alert('爬虫已停止！');
        loadStatus();
    })
    .catch(error => console.error('Error:', error));
}

// 加载关键词列表
function loadKeywords() {
    fetch('/api/keywords')
        .then(response => response.json())
        .then(data => {
            const tbody = document.getElementById('keywordsTable');
            tbody.innerHTML = '';
            
            data.forEach(keyword => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${keyword.id}</td>
                    <td>${keyword.zh_key}</td>
                    <td>${keyword.en_key}</td>
                    <td>${keyword.priority}</td>
                    <td>
                        <button class="btn btn-sm btn-danger delete-keyword" data-id="${keyword.id}">删除</button>
                    </td>
                `;
                tbody.appendChild(tr);
            });

            // 添加删除事件监听
            document.querySelectorAll('.delete-keyword').forEach(btn => {
                btn.addEventListener('click', function() {
                    deleteKeyword(this.getAttribute('data-id'));
                });
            });
        })
        .catch(error => console.error('Error:', error));
}

// 保存关键词
function saveKeyword() {
    const keyword = {
        zh_key: document.getElementById('zhKey').value.trim(),
        en_key: document.getElementById('enKey').value.trim(),
        priority: parseInt(document.getElementById('priority').value)
    };

    // 验证输入
    if (!keyword.zh_key || !keyword.en_key) {
        alert('中文关键词和英文关键词都不能为空！');
        return;
    }

    fetch('/api/keywords', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(keyword)
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('保存关键词失败，请重试！');
        }
        return response.json();
    })
    .then(data => {
        // 关闭模态框
        const modal = bootstrap.Modal.getInstance(document.getElementById('keywordModal'));
        modal.hide();
        
        // 清空表单
        document.getElementById('zhKey').value = '';
        document.getElementById('enKey').value = '';
        document.getElementById('priority').value = '0';
        
        // 重新加载关键词列表
        loadKeywords();
        
        // 显示成功消息
        alert('关键词保存成功！');
    })
    .catch(error => {
        console.error('Error:', error);
        alert(error.message);
    });
}

// 删除关键词
function deleteKeyword(id) {
    if (confirm('确定要删除此关键词吗？')) {
        fetch(`/api/keywords/${id}`, {
            method: 'DELETE'
        })
        .then(response => response.json())
        .then(data => {
            loadKeywords();
        })
        .catch(error => console.error('Error:', error));
    }
}

// 加载产品列表
function loadProducts(page, limit = 100) {
    const offset = page * limit;
    
    fetch(`/api/results?limit=${limit}&offset=${offset}`)
        .then(response => response.json())
        .then(data => {
            const tbody = document.getElementById('productsTable');
            tbody.innerHTML = '';
            
            data.forEach(product => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${product.id}</td>
                    <td><a href="${product.url}" target="_blank">${truncateText(product.url, 50)}</a></td>
                    <td>${getStatusText(product.status)}</td>
                    <td>${product.zh_key || '-'}</td>
                `;
                tbody.appendChild(tr);
            });

            // 更新分页
            updatePagination('productsPagination', page, Math.ceil(data.length / limit), loadProducts);
        })
        .catch(error => console.error('Error:', error));
}

// 加载商家信息
function loadSellers(page, query = '', limit = 100) {
    const offset = page * limit;
    let url = `/api/sellers?limit=${limit}&offset=${offset}`;
    
    if (query) {
        url += `&query=${encodeURIComponent(query)}`;
    }
    
    fetch(url)
        .then(response => response.json())
        .then(data => {
            const tbody = document.getElementById('sellersTable');
            tbody.innerHTML = '';
            
            data.forEach(seller => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${seller.id}</td>
                    <td>${seller.seller_id}</td>
                    <td>${seller.name || '-'}</td>
                    <td>${truncateText(seller.address || '-', 50)}</td>
                    <td>${seller.trn || '-'}</td>
                `;
                tbody.appendChild(tr);
            });

            // 更新分页
            updatePagination('sellersPagination', page, Math.ceil(data.length / limit), function(p) {
                loadSellers(p, query, limit);
            });
        })
        .catch(error => console.error('Error:', error));
}

// 加载日志
function loadLogs() {
    const logLevel = document.getElementById('logLevelSelect').value;
    const lines = document.getElementById('logLines').value;
    
    fetch(`/api/logs?level=${logLevel}&lines=${lines}`)
        .then(response => response.json())
        .then(data => {
            const logContent = document.getElementById('logContent');
            
            if (data.logs && data.logs.length > 0) {
                // 格式化日志并高亮不同级别
                const formattedLogs = data.logs.map(log => {
                    let levelClass = '';
                    if (log.includes('[INFO]')) {
                        levelClass = 'log-info';
                    } else if (log.includes('[WARN]')) {
                        levelClass = 'log-warn';
                    } else if (log.includes('[ERROR]')) {
                        levelClass = 'log-error';
                    }
                    
                    // 尝试提取并格式化时间戳
                    const timeMatch = log.match(/^\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}/);
                    if (timeMatch) {
                        const timestamp = timeMatch[0];
                        const restOfLog = log.substring(timestamp.length);
                        return `<span class="log-time">${timestamp}</span><span class="${levelClass}">${restOfLog}</span>`;
                    }
                    
                    return `<span class="${levelClass}">${log}</span>`;
                }).join('\n');
                
                logContent.innerHTML = formattedLogs;
            } else {
                logContent.innerHTML = '暂无日志数据';
            }
            
            // 滚动到底部
            logContent.scrollTop = logContent.scrollHeight;
        })
        .catch(error => {
            console.error('Error:', error);
            document.getElementById('logContent').innerHTML = '加载日志失败: ' + error.message;
        });
}

// 加载配置
function loadConfig() {
    fetch('/api/config')
        .then(response => response.json())
        .then(data => {
            document.getElementById('appId').value = data.app_id;
            document.getElementById('hostId').value = data.host_id;
            document.getElementById('domain').value = data.domain;
            document.getElementById('searchPriority').value = data.search_priority;
            
            // 加载循环次数设置
            if (data.loop_all !== undefined) document.getElementById('allLoopCount').value = data.loop_all;
            if (data.loop_search !== undefined) document.getElementById('searchLoopCount').value = data.loop_search;
            if (data.loop_product !== undefined) document.getElementById('productLoopCount').value = data.loop_product;
            if (data.loop_seller !== undefined) document.getElementById('sellerLoopCount').value = data.loop_seller;
        })
        .catch(error => console.error('Error:', error));
}

// 保存配置
function saveConfig() {
    const config = {
        app_id: parseInt(document.getElementById('appId').value),
        host_id: parseInt(document.getElementById('hostId').value),
        domain: document.getElementById('domain').value,
        search_enabled: document.getElementById('searchToggle').checked,
        product_enabled: document.getElementById('productToggle').checked,
        seller_enabled: document.getElementById('sellerToggle').checked,
        search_priority: parseInt(document.getElementById('searchPriority').value),
        proxy_enabled: false,
        proxy_socks5: []
    };

    fetch('/api/config', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(config)
    })
    .then(response => response.json())
    .then(data => {
        alert('配置已保存！');
    })
    .catch(error => console.error('Error:', error));
}

// 加载Cookie
function loadCookie() {
    fetch('/api/cookie')
        .then(response => response.json())
        .then(data => {
            document.getElementById('cookieText').value = data.cookie;
        })
        .catch(error => console.error('Error:', error));
}

// 保存Cookie
function saveCookie() {
    const cookieData = {
        cookie: document.getElementById('cookieText').value
    };

    fetch('/api/cookie', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(cookieData)
    })
    .then(response => response.json())
    .then(data => {
        alert('Cookie已保存！');
    })
    .catch(error => console.error('Error:', error));
}

// 更新分页
function updatePagination(elementId, currentPage, totalPages, callback) {
    const pagination = document.getElementById(elementId);
    pagination.innerHTML = '';
    
    // 前一页按钮
    const prevLi = document.createElement('li');
    prevLi.classList.add('page-item');
    if (currentPage === 0) {
        prevLi.classList.add('disabled');
    }
    const prevLink = document.createElement('a');
    prevLink.classList.add('page-link');
    prevLink.href = '#';
    prevLink.textContent = '上一页';
    prevLink.addEventListener('click', function(e) {
        e.preventDefault();
        if (currentPage > 0) {
            callback(currentPage - 1);
        }
    });
    prevLi.appendChild(prevLink);
    pagination.appendChild(prevLi);
    
    // 页码按钮
    const maxVisiblePages = 5;
    let startPage = Math.max(0, currentPage - Math.floor(maxVisiblePages / 2));
    let endPage = Math.min(totalPages - 1, startPage + maxVisiblePages - 1);
    
    if (endPage - startPage + 1 < maxVisiblePages) {
        startPage = Math.max(0, endPage - maxVisiblePages + 1);
    }
    
    for (let i = startPage; i <= endPage; i++) {
        const pageLi = document.createElement('li');
        pageLi.classList.add('page-item');
        if (i === currentPage) {
            pageLi.classList.add('active');
        }
        const pageLink = document.createElement('a');
        pageLink.classList.add('page-link');
        pageLink.href = '#';
        pageLink.textContent = i + 1;
        pageLink.addEventListener('click', function(e) {
            e.preventDefault();
            callback(i);
        });
        pageLi.appendChild(pageLink);
        pagination.appendChild(pageLi);
    }
    
    // 下一页按钮
    const nextLi = document.createElement('li');
    nextLi.classList.add('page-item');
    if (currentPage >= totalPages - 1) {
        nextLi.classList.add('disabled');
    }
    const nextLink = document.createElement('a');
    nextLink.classList.add('page-link');
    nextLink.href = '#';
    nextLink.textContent = '下一页';
    nextLink.addEventListener('click', function(e) {
        e.preventDefault();
        if (currentPage < totalPages - 1) {
            callback(currentPage + 1);
        }
    });
    nextLi.appendChild(nextLink);
    pagination.appendChild(nextLi);
}

// 获取状态文本
function getStatusText(status) {
    switch (status) {
        case 0: return '未搜索';
        case 1: return '准备检查';
        case 2: return '检查结束';
        case 3: return '没有商家';
        case 4: return '其他错误';
        default: return `未知(${status})`;
    }
}

// 截断文本
function truncateText(text, maxLength) {
    if (text.length <= maxLength) {
        return text;
    }
    return text.substring(0, maxLength) + '...';
} 