package models

import "time"

// CookieStatus Cookie 状态枚举（用于计算状态，非存储）
type CookieStatus int

const (
	CookieStatusActive  CookieStatus = 0 // 可用
	CookieStatusExpired CookieStatus = 1 // 已过期
	CookieStatusFrozen  CookieStatus = 2 // 冷冻中
)

// Cookie Cookie 数据模型
type Cookie struct {
	ID            int64      `db:"id"`
	Platform      string     `db:"platform"`       // 平台名称：youtube/bilibili/tiktok
	Name          string     `db:"name"`           // Cookie 名称/标识
	Content       string     `db:"content"`        // Cookie 内容（Netscape 格式）
	ExpireAt      *time.Time `db:"expire_at"`      // 过期时间
	FrozenUntil   *time.Time `db:"frozen_until"`   // 冷冻结束时间
	FreezeSeconds int        `db:"freeze_seconds"` // 使用后冷冻秒数
	LastUsedAt    *time.Time `db:"last_used_at"`   // 上次使用时间
	UseCount      int        `db:"use_count"`      // 使用次数
	SuccessCount  int        `db:"success_count"`  // 成功使用次数
	FailCount     int        `db:"fail_count"`     // 失败使用次数
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

// CookieFilter Cookie 查询过滤条件
type CookieFilter struct {
	Platform      *string       // 可选：平台过滤
	Status        *CookieStatus // 可选：状态过滤（通过计算过期和冷冻判断）
	Page          int
	PageSize      int
	OnlyAvailable bool // 可选：只查询可用的 Cookie
}

// CookieResult Cookie 查询结果
type CookieResult struct {
	Total    int64
	Page     int
	PageSize int
	Items    []Cookie
}

// IsAvailable 检查 Cookie 是否可用（未过期且未冷冻）
func (c *Cookie) IsAvailable() bool {
	now := time.Now()

	// 检查过期
	if c.ExpireAt != nil && now.After(*c.ExpireAt) {
		return false
	}

	// 检查冷冻
	if c.FrozenUntil != nil && now.Before(*c.FrozenUntil) {
		return false
	}

	return true
}

// toLocalTime 将数据库时间（被 Go 驱动误解析为 UTC）转换为实际的本地时间
// 数据库存储的是 timestamp without time zone，存的是本地时间值，
// 但 Go 驱动会将其解析为 UTC 时间，需要重新解释为本地时间
func toLocalTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	// 提取时间的年月日时分秒纳秒，在本地时区重新创建
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
}

// IsExpired 检查 Cookie 是否已过期
func (c *Cookie) IsExpired() bool {
	if c.ExpireAt == nil {
		return false
	}
	dbTime := toLocalTime(c.ExpireAt)
	return time.Now().After(dbTime)
}

// IsFrozen 检查 Cookie 是否正在冷冻中
func (c *Cookie) IsFrozen() bool {
	if c.FrozenUntil == nil {
		return false
	}
	dbTime := toLocalTime(c.FrozenUntil)
	return time.Now().Before(dbTime)
}

// GetEffectiveStatus 获取有效状态（根据过期时间和冷冻时间动态计算）
func (c *Cookie) GetEffectiveStatus() CookieStatus {
	// 优先检查过期（过期状态优先级最高）
	if c.IsExpired() {
		return CookieStatusExpired
	}

	// 其次检查冷冻
	if c.IsFrozen() {
		return CookieStatusFrozen
	}

	return CookieStatusActive
}
