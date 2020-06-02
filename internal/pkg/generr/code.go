package generr

type mErr struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

var (
	ParseParam  = &mErr{400, "参数错误"}
	ServerError = &mErr{500, "服务错误"}
)

var (
	SignMiss     = &mErr{601, "s参数缺失"}
	SignNotMatch = &mErr{602, "s不匹配"}
	TimestampErr = &mErr{603, "t参数错误"}
	TimestampOut = &mErr{604, "t超时"}
	ReadDB       = &mErr{698, "读数据库错误"}
	UpdateDB     = &mErr{699, "更新数据库错误"}

	SugarNoTargetUser = &mErr{701, "无目标用户信息"}
	SugarNoToken      = &mErr{701, "token缺失"}
	SugarWrongToken   = &mErr{702, "token不匹配"}
	SugarNoFile       = &mErr{703, "文件缺失"}
	SugarWrongFile    = &mErr{704, "未知文件"}
	SugarRepeatFile   = &mErr{705, "文件重复"}
	SugarFormFile     = &mErr{706, "获取文件错误"}
	SugarSaveFile     = &mErr{707, "存储文件错误"}

	DestructAmountError = &mErr{801, "销毁金额服务错误"}
	CnyToSieErr         = &mErr{802, "CNY转换SIE错误"}
	SusdToSieErr        = &mErr{803, "SUSD转换SIE错误"}

	BalanceNotEnough = &mErr{901, "余额不足"}
)
