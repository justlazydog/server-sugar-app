package generr

type mErr struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

var (
	ParseParam = &mErr{601, "参数错误"}
	ReadDB     = &mErr{602, "读数据库错误"}
	UpdateDB   = &mErr{603, "更新数据库错误"}
)

var (
	SugarNoTargetUser = &mErr{701, "无目标用户信息"}
	SugarNoToken      = &mErr{701, "token缺失"}
	SugarWrongToken   = &mErr{702, "token不匹配"}
	SugarNoFile       = &mErr{703, "文件缺失"}
	SugarWrongFile    = &mErr{704, "未知文件"}
	SugarRepeatFile   = &mErr{705, "文件重复"}
	SugarFormFile     = &mErr{706, "获取文件错误"}
	SugarSaveFile     = &mErr{707, "存储文件错误"}
)
