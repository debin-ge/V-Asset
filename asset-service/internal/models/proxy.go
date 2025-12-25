package models

import "time"

// ProxyStatus 代理状态枚举
type ProxyStatus int

const (
	ProxyStatusActive   ProxyStatus = 0 // 可用
	ProxyStatusInactive ProxyStatus = 1 // 不可用
	ProxyStatusChecking ProxyStatus = 2 // 检查中
)

// ProxyProtocol 代理协议类型
type ProxyProtocol string

const (
	ProxyProtocolHTTP   ProxyProtocol = "http"
	ProxyProtocolHTTPS  ProxyProtocol = "https"
	ProxyProtocolSOCKS5 ProxyProtocol = "socks5"
)

// Proxy 代理数据模型
type Proxy struct {
	ID              int64         `db:"id"`
	IP              string        `db:"ip"`
	Port            int           `db:"port"`
	Username        *string       `db:"username"`          // 认证用户名（可选）
	Password        *string       `db:"password"`          // 认证密码（可选）
	Protocol        ProxyProtocol `db:"protocol"`          // 协议类型：http/https/socks5
	Region          *string       `db:"region"`            // 地区标签
	Status          ProxyStatus   `db:"status"`            // 健康状态
	LastCheckAt     *time.Time    `db:"last_check_at"`     // 上次健康检查时间
	LastCheckResult *string       `db:"last_check_result"` // 上次检查结果
	SuccessCount    int           `db:"success_count"`     // 成功使用次数
	FailCount       int           `db:"fail_count"`        // 失败使用次数
	LastUsedAt      *time.Time    `db:"last_used_at"`      // 上次使用时间
	CreatedAt       time.Time     `db:"created_at"`
	UpdatedAt       time.Time     `db:"updated_at"`
}

// ProxyFilter 代理查询过滤条件
type ProxyFilter struct {
	Status   *ProxyStatus   // 可选：状态过滤
	Protocol *ProxyProtocol // 可选：协议过滤
	Region   *string        // 可选：地区过滤
	Page     int
	PageSize int
}

// ProxyResult 代理查询结果
type ProxyResult struct {
	Total    int64
	Page     int
	PageSize int
	Items    []Proxy
}

// GetURL 获取代理完整 URL
func (p *Proxy) GetURL() string {
	var url string
	username := ""
	password := ""
	if p.Username != nil {
		username = *p.Username
	}
	if p.Password != nil {
		password = *p.Password
	}

	if username != "" && password != "" {
		url = string(p.Protocol) + "://" + username + ":" + password + "@" + p.IP + ":" + itoa(p.Port)
	} else {
		url = string(p.Protocol) + "://" + p.IP + ":" + itoa(p.Port)
	}
	return url
}

// itoa 简单的整数转字符串
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var buf [10]byte
	pos := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}

	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}

	if neg {
		pos--
		buf[pos] = '-'
	}

	return string(buf[pos:])
}
