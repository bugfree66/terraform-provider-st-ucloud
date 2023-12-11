package api

const (
	ERR_CODE_RATE_LIMIT = 153
	ERR_CODE_TOO_OFTEN  = 44025
)

func Retryable(code int) bool {
	switch code {
	case ERR_CODE_RATE_LIMIT,
		ERR_CODE_TOO_OFTEN:
		return true
	default:
		return false
	}
}
