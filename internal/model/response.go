package model

type ListBossCreditRsp struct {
	OpenID    string  `json:"open_id"`
	AllCredit float64 `json:"all_credit"`
	Num       int     `json:"num"`
}
