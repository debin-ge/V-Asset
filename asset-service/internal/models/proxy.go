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
	ID                   int64         `db:"id"`
	Host                 *string       `db:"host"`
	IP                   string        `db:"ip"`
	Port                 int           `db:"port"`
	Username             *string       `db:"username"` // 认证用户名（可选）
	Password             *string       `db:"password"` // 认证密码（可选）
	Protocol             ProxyProtocol `db:"protocol"` // 协议类型：http/https/socks5
	Region               *string       `db:"region"`   // 地区标签
	Priority             int           `db:"priority"`
	PlatformTags         *string       `db:"platform_tags"`
	Remark               *string       `db:"remark"`
	Status               ProxyStatus   `db:"status"`            // 健康状态
	LastCheckAt          *time.Time    `db:"last_check_at"`     // 上次健康检查时间
	LastCheckResult      *string       `db:"last_check_result"` // 上次检查结果
	SuccessCount         int           `db:"success_count"`     // 成功使用次数
	FailCount            int           `db:"fail_count"`        // 失败使用次数
	LastUsedAt           *time.Time    `db:"last_used_at"`      // 上次使用时间
	CooldownUntil        *time.Time    `db:"cooldown_until"`
	ConsecutiveFailCount int           `db:"consecutive_fail_count"`
	RiskScore            int           `db:"risk_score"`
	LastErrorCategory    *string       `db:"last_error_category"`
	LastFailAt           *time.Time    `db:"last_fail_at"`
	MaxConcurrent        int           `db:"max_concurrent"`
	ActiveTaskCount      int           `db:"active_task_count"`
	DeletedAt            *time.Time    `db:"deleted_at"`
	CreatedAt            time.Time     `db:"created_at"`
	UpdatedAt            time.Time     `db:"updated_at"`
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
	host := p.IP
	if p.Host != nil && *p.Host != "" {
		host = *p.Host
	}

	if username != "" && password != "" {
		url = string(p.Protocol) + "://" + username + ":" + password + "@" + host + ":" + itoa(p.Port)
	} else {
		url = string(p.Protocol) + "://" + host + ":" + itoa(p.Port)
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
