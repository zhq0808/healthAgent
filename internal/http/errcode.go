package http

// 业务错误码。约定：0 成功；其余按 HTTP 语义分段，便于前端与日志排查。
// Phase 1/2 的具体业务错误在此集中登记，避免 handler 里散落 magic number。
const (
	CodeOK = 0

	CodeBadRequest = 40000 // 通用参数错误
	CodeNotFound   = 40400 // 资源不存在
	CodeMethodNA   = 40500 // 方法不允许

	CodeInternal = 50000 // 内部错误

	// Phase 1 录入相关（占位，实现时启用）：
	// CodeDraftNotFound    = 40401
	// CodeDraftAlreadyDone = 40901
	// CodeUnsupportedUnit  = 42201
	// CodeLLMInvalidJSON   = 50201
)
