package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tengfei-xy/go-log"
)

type sellerStruct struct {
	seller_id    string
	url          string
	businessName string
	trn          string
	address      string
	primary_id   string
	trn_status   int
	all_status   int
}

const MYSQL_SELLER_STATUS_TRN_OK int = 1
const MYSQL_SELLER_STATUS_TRN_NO int = 2
const MYSQL_SELLER_STATUS_TRN_OTHER int = 3
const MYSQL_SELLER_STATUS_TRN_SPECIAL int = 4

const MYSQL_SELLER_STATUS_INFO_INSERT int = MYSQL_PRODUCT_STATUS_INSERT
const MYSQL_SELLER_STATUS_INFO_OK int = 1
const MYSQL_SELLER_STATUS_INFO_ALL_NO_NAME int = 2
const MYSQL_SELLER_STATUS_INFO_ALL_NO_ADDRESS int = 3
const MYSQL_SELLER_STATUS_INFO_ALL_NO_TRN int = 4

func (seller *sellerStruct) prepare() {
	app.update(MYSQL_APPLICATION_STATUS_SELLER)
	log.Infof("------------------------")
	log.Infof("3. 开始 根据商家页获取商家信息")
}
func (seller *sellerStruct) over() {
	log.Infof("3. 结束 根据商家页获取商家信息")
	log.Infof("------------------------")
}
func (seller *sellerStruct) start() error {
	_, err := app.db.Exec("UPDATE seller SET app_id = ? WHERE all_status = ? and (app_id=? or app_id=?) LIMIT 100", app.Basic.App_id, MYSQL_SELLER_STATUS_INFO_INSERT, 0, app.Basic.App_id)
	if err != nil {
		log.Errorf("更新seller表失败,%v", err)
		return err
	}
	return nil
}
func (seller *sellerStruct) main() error {

	if !app.Exec.Enable.Seller {
		log.Warn("跳过 获取商家信息")
		return nil
	}
	if app.Exec.Loop.Seller == app.Exec.Loop.seller_time {
		log.Warn("已经达到执行次数 获取商家信息")
		return nil
	}

	seller.prepare()

	if app.Exec.Loop.Seller == 0 {
		log.Info("循环次数无限")
	} else {
		log.Infof("循环次数剩余:%d", app.Exec.Loop.Seller-app.Exec.Loop.seller_time)
	}
	app.Exec.Loop.seller_time++

	if err := seller.start(); err != nil {
		return err
	}

	row, err := app.db.Query("select id,seller_id from seller where all_status =? and app_id=?", MYSQL_SELLER_STATUS_INFO_INSERT, app.Basic.App_id)
	switch err {
	case nil:
		break
	case sql.ErrNoRows:
		log.Warnf("指定的app_id:%d,没有需要处理的商家信息", app.Basic.App_id)
		return nil
	default:
		log.Error(err)
		return err

	}
	for row.Next() {
		// 重置字段,避免上一商家数据残留(曾导致 A 家 TRN 记到 B 家)
		seller.businessName = ""
		seller.trn = ""
		seller.address = ""
		seller.trn_status = 0
		seller.all_status = MYSQL_SELLER_STATUS_INFO_OK

		if err := row.Scan(&seller.primary_id, &seller.seller_id); err != nil {
			log.Error(err)
			continue
		}
		seller.url = fmt.Sprintf("https://%s/sp?ie=UTF8&seller=%s", app.Domain, seller.seller_id)

		if err := robot.IsAllow(userAgent, seller.url); err != nil {
			log.Errorf("%v", err)
			continue
		}

		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			lastErr = seller.request()
			if lastErr == nil {
				break
			}
			log.Error(lastErr)
			sleep(30)
		}
		if lastErr != nil {
			log.Errorf("请求失败已重试3次,跳过 商家ID:%s err:%v", seller.seller_id, lastErr)
			seller.all_status = MYSQL_SELLER_STATUS_INFO_ALL_NO_NAME
			if err := seller.update(); err != nil {
				log.Error(err)
			}
			continue
		}

		seller.trnCheck()
		seller.addressCheck()
		seller.nameCheck()
		if err := seller.update(); err != nil {
			log.Error(err)
			continue
		}

	}

	seller.over()
	return nil
}

// 作用: 根据 seller.url 请求商家信息
// 举例: 根据 https://www.amazon.co.uk/sp?ie=UTF8&seller=A272CUATTYX3C4 请求商家信息
func (seller *sellerStruct) request() error {

	log.Infof("请求链接 %s", seller.url)

	client := get_client()
	req, err := http.NewRequest("GET", seller.url, nil)
	if err != nil {
		return err
	}
	applyHeaders(req)
	if _, err := app.get_cookie(); err != nil {
		log.Error(err)
	} else {
		req.Header.Set("Cookie", app.cookie)
	}
	req.Header.Set("Referer", fmt.Sprintf("https://%s/s?k=%s&ref=nb_sb_noss", app.Domain, "custom"))

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("内部错误:%v", err)
		return err

	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		break
	case 404:
		return ERROR_NOT_404
	case 503:
		return ERROR_NOT_503
	default:
		return fmt.Errorf("状态码:%d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("内部错误:%v", err)
	}

	sellerTxt := doc.Find("div#page-section-detail-seller-info").Find("span").Text()

	seller.all_status = MYSQL_SELLER_STATUS_INFO_OK

	// 逐行解析 span 文本: 每行形如 "Business Name: XXX" 或 "XXX"
	// 兼容 Amazon 两种排版(标签与值同行 / 标签单独一行、值在下一行)
	rawLines := strings.Split(sellerTxt, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, l := range rawLines {
		l = strings.TrimSpace(l)
		if l != "" {
			lines = append(lines, l)
		}
	}
	labels := map[string]*string{
		"Business Name:":         &seller.businessName,
		"Trade Register Number:": &seller.trn,
		"Business Address:":      &seller.address,
		"VAT Number:":            nil, // 不提取,仅作分隔符
	}
parse:
	for i := 0; i < len(lines); i++ {
		for label, dst := range labels {
			if !strings.HasPrefix(lines[i], label) {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(lines[i], label))
			if value == "" {
				// 值在下一行
				if i+1 < len(lines) && !isSellerLabel(lines[i+1]) {
					value = strings.TrimSpace(lines[i+1])
					i++
				}
			}
			if dst != nil && value != "" {
				*dst = value
				log.Infof("解析到 %s %s", label, value)
			}
			continue parse
		}
		// 未匹配标签的行: 若是地址的续行(标签后的第二行起),续拼进地址
		if len(seller.address) > 0 && i > 0 && !strings.Contains(lines[i-1], "Address") {
			seller.address += " " + lines[i]
		}
	}

	return nil
}

// 判断某行是否为卖家信息标签行
func isSellerLabel(line string) bool {
	for _, label := range []string{"Business Name:", "Business Type:", "Trade Register Number:", "Business Address:", "VAT Number:"} {
		if strings.HasPrefix(line, label) {
			return true
		}
	}
	return false
}

// 作用: 检查TRN
func (seller *sellerStruct) trnCheck() {
	seller.trn = strings.ReplaceAll(seller.trn, "(1-1)", "")
	if len(seller.trn) == 0 {
		log.Warnf("检查结果 TRN为空")
		seller.trn_status = MYSQL_SELLER_STATUS_TRN_NO
		return
	} else if len(seller.trn) < 18 {
		log.Warnf("检查结果 TRN: %s (非中国)", seller.trn)
		seller.trn_status = MYSQL_SELLER_STATUS_TRN_OTHER
		return
	} else if len(seller.trn) > 18 {
		log.Warnf("检查结果 TRN: %s (非中国)", seller.trn)
		seller.trn = ""
		seller.trn_status = MYSQL_SELLER_STATUS_TRN_OTHER
		return
	}
	if seller.trn[0] != '9' {
		log.Warnf("检查结果 TRN: %s (18位长,非9开头)", seller.trn)
		seller.trn_status = MYSQL_SELLER_STATUS_TRN_SPECIAL
		return
	}
	log.Infof("查找结果 TRN: %s(中国)", seller.trn)
	seller.trn_status = MYSQL_SELLER_STATUS_TRN_OK
	return
}

func (seller *sellerStruct) addressCheck() {
	if len(seller.address) == 0 {
		log.Errorf("检查结果 地址为空")
		return
	}
	log.Infof("查找结果 地址: %s", seller.address)
}
func (seller *sellerStruct) nameCheck() {
	if len(seller.businessName) == 0 {
		log.Errorf("检查结果 商家名称为空")
		return
	}
	log.Infof("查找结果 商家名称: %s", seller.businessName)
}

// 汇总三个字段的缺失情况,优先级: 名称缺失 > 地址缺失 > TRN缺失(与视图语义一致)
func (seller *sellerStruct) calcAllStatus() int {
	if len(seller.businessName) == 0 {
		return MYSQL_SELLER_STATUS_INFO_ALL_NO_NAME
	}
	if len(seller.address) == 0 {
		return MYSQL_SELLER_STATUS_INFO_ALL_NO_ADDRESS
	}
	if len(seller.trn) == 0 {
		return MYSQL_SELLER_STATUS_INFO_ALL_NO_TRN
	}
	return MYSQL_SELLER_STATUS_INFO_OK
}
func (seller *sellerStruct) update() error {
	_, err := app.db.Exec("update seller set trn_status=?,trn=?,name=?,address=?,all_status=? where id=? and app_id=?", seller.trn_status, seller.trn, seller.businessName, seller.address, seller.calcAllStatus(), seller.primary_id, app.Basic.App_id)
	return err
}
