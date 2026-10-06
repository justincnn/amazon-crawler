package main

import "testing"

func TestGetRobotFromTxt_EmptyDisallow(t *testing.T) {
	// robots.txt 常见空规则行("Disallow: " 后无路径, RFC 9309 表示不限制)
	txt := "User-agent: *\nDisallow: \nDisallow: /gp/\nAllow: /dp/\n"
	r := GetRobotFromTxt(txt)

	for _, ua := range []string{userAgent, "other-bot"} {
		if err := r.IsAllow(ua, "https://www.amazon.com/s?k=x"); err != nil {
			t.Errorf("普通页面被拒(空Disallow被当成禁止一切): %v", err)
		}
	}

	// /gp/ 路径应被禁用
	for _, ua := range []string{userAgent, "other-bot"} {
		ru := r.IsAllow(ua, "https://www.amazon.com/gp/help")
		if ru == nil {
			t.Errorf("/gp/ 路径应被 Disallow 拦截")
		}
	}

	// Allow 优先于 Disallow
	if err := r.IsAllow(userAgent, "https://www.amazon.com/"); err != nil {
		t.Errorf("Allow: / 应覆盖 Disallow: /gp/")
	}
}

func TestCalcAllStatus(t *testing.T) {
	s := &sellerStruct{businessName: "ACME", address: "addr", trn: "91330100MA27X0000X"}
	if got := s.calcAllStatus(); got != MYSQL_SELLER_STATUS_INFO_OK {
		t.Errorf("全字段完整应得 OK=%d, got %d", MYSQL_SELLER_STATUS_INFO_OK, got)
	}
	s.businessName = ""
	if got := s.calcAllStatus(); got != MYSQL_SELLER_STATUS_INFO_ALL_NO_NAME {
		t.Errorf("缺名称应得 NO_NAME=%d, got %d", MYSQL_SELLER_STATUS_INFO_ALL_NO_NAME, got)
	}
	s.businessName = "京"
	s.address = ""
	if got := s.calcAllStatus(); got != MYSQL_SELLER_STATUS_INFO_ALL_NO_ADDRESS {
		t.Errorf("缺地址应得 NO_ADDRESS=%d, got %d", MYSQL_SELLER_STATUS_INFO_ALL_NO_ADDRESS, got)
	}
	s.address = "addr"
	s.trn = ""
	if got := s.calcAllStatus(); got != MYSQL_SELLER_STATUS_INFO_ALL_NO_TRN {
		t.Errorf("缺税号应得 NO_TRN=%d, got %d", MYSQL_SELLER_STATUS_INFO_ALL_NO_TRN, got)
	}
}