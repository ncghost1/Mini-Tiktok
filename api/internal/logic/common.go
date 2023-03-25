package logic

const (
	STATUS_SUCCESS          = "0"
	STATUS_FAIL             = "1"
	STATUS_FAIL_TOKEN_MSG   = "Token is invalid"
	STATUS_FAIL_TOOLONG_MSG = "Username or Password must less than 32 characters"
	STATUS_SUCCESS_MSG      = "OK"
	STATUS_FAIL_PARAM_MSG   = "Request parameter error"
	USER_NO_LOGIN           = "0" // 需保证不出现 id 为 0 的用户
	STATUS_FAIL_FOLLOW_SELF = "Follow yourself is not allowed"
	FILE_EMPTY_ERROR        = "upload file is empty"
	FILE_TYPE_ERROR         = "upload file type error"
	MP4_TYPE                = "video/mp4"
)
