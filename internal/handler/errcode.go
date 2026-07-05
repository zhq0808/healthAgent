package handler

// 业务错误码。约定：0 成功；其余按 HTTP 语义分段，便于前端与日志排查。
const (
	CodeOK = 0

	CodeBadRequest = 40000 // 通用参数错误
	CodeNotFound   = 40400 // 资源不存在
	CodeMethodNA   = 40500 // 方法不允许

	CodeInternal = 50000 // 内部错误
)
