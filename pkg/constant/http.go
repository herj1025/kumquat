package constant

const (
	// OK 请求成功
	OK = 0

	// BadRequest 请求参数错误，用于客户端提交的数据格式不正确时
	BadRequest = 4000
	// MissingParam 缺少必要参数，用于客户端未提交必填字段时
	MissingParam = 4001
	// InvalidParam 参数格式错误，用于提交的字段值不满足类型或范围要求时
	InvalidParam = 4002

	// Unauthorized 未认证，用于请求未携带有效身份凭证时
	Unauthorized = 4010
	// TokenExpired token 过期，用于身份凭证已超过有效期时
	TokenExpired = 4011
	// TokenInvalid token 无效，用于身份凭证格式错误或被篡改时
	TokenInvalid = 4012

	// Forbidden 无权限，用于用户已认证但没有操作权限时
	Forbidden = 4030

	// NotFound 资源不存在，用于请求的目标资源未找到时
	NotFound = 4040
	// Conflict 资源冲突，用于创建的资源已存在或状态冲突时（如重复提交）
	Conflict = 4090

	// RateLimited 请求频率限制，用于客户端请求过快被限流时
	RateLimited = 4290

	// InternalError 服务器内部错误，用于代码逻辑异常、空指针、序列化失败等自身 bug
	InternalError = 5000
	// ServiceUnavail 服务不可用，用于数据库断开、第三方 API 超时等外部依赖故障
	ServiceUnavail = 5001
)
