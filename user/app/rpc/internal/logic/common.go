package logic

const (
	STATUS_SUCCESS            = "0"
	STATUS_SUCCESS_MSG        = "OK"
	STATUS_FAIL               = "1"
	STATUS_FAIL_PARAM_MSG     = "Param incorrect"
	STATUS_USER_EXISTS_MSG    = "Username already exists"
	STATUS_USER_NOTEXIST_MSG  = "User not exist"
	STATUS_WRONG_PASSWORD_MSG = "Wrong Password"
	COUNT_NOT_FOUND           = int64(-1)
	OP_FOLLOW                 = "1"
	OP_CANCEL_FOLLOW          = "2"
)
