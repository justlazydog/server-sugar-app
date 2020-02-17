package model

import "time"

// 用户信息表结构
type User struct {
	ID        int     `json:"id"`         // 数据ID
	UID       string  `json:"-"`          // 用户uid
	OpenID    string  `json:"open_id"`    // 用户Open_id
	OrderID   string  `json:"order_id"`   // 挂单ID
	Amount    float64 `json:"amount"`     // 销毁金额
	Credit    float64 `json:"credit"`     // 用户积分
	Multiple  float64 `json:"multiple"`   // 倍数
	Flag      uint8   `json:"flag"`       // 1-线下 2-线上
	CreatedAt int64   `json:"created_at"` // 创建时间（此处为Unix时间戳）
}

type Boss struct {
	ID        int     `json:"id"`         // 数据ID
	UID       string  `json:"-"`          // 店主uid
	OpenID    string  `json:"open_id"`    // 店主Open_id
	OrderID   string  `json:"order_id"`   // 挂单ID
	Amount    float64 `json:"amount"`     // 销毁金额
	Credit    float64 `json:"credit"`     // 商户积分
	Multiple  float64 `json:"multiple"`   // 倍数
	Flag      uint8   `json:"flag"`       // 1-线下 2-线上
	CreatedAt int64   `json:"created_at"` // 创建时间（此处为Unix时间戳）
}

type Sugar struct {
	CreateTime   time.Time `json:"-"`
	Sugar        float64   `json:"sugar"`         // 当日糖果发放量
	Currency     float64   `json:"currency"`      // 当日流通量
	RealCurrency float64   `json:"real_currency"` // 当日实际流通量
	ShopSIE      float64   `json:"shop_sie"`      // 当前商户SIE
	ShopUsedSIE  float64   `json:"shop_used_sie"` // 当前商户销毁SIE
	AccountIn    float64   `json:"account_in"`    // 差值账户in
	AccountOut   float64   `json:"account_out"`   // 差值账户out
}

type UserSugar struct {
	UID              string  `json:"uid"`                // 用户ID
	UserSugarAmount  float64 `json:"user_sugar_amount"`  // 用户糖果数量
	UserPossessForce float64 `json:"user_possess_force"` // 用户持币算力
	UserInviteForce  float64 `json:"user_invite_force"`  // 用户邀请算力
	UserAmount       float64 `json:"user_amount"`        // 用户持币量
	UserFrozen       float64 `json:"user_frozen"`        // 用户冻结金额
	UserCredit       float64 `json:"user_credit"`        // 用户积分
}