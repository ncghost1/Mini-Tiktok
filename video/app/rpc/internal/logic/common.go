package logic

const (
	STATUS_SUCCESS        = "0"
	STATUS_SUCCESS_MSG    = "OK"
	STATUS_FAIL           = "1"
	STATUS_FAIL_PARAM_MSG = "Request parameter error"
	COMMENT_UPDATE        = "1"
	COMMENT_DELETE        = "2"
	FAVORITE_UPDATE       = "1"
	FAVORITE_DELETE       = "2"
	OP_INSERT             = "insert"
	OP_DELETE             = "delete"
	MODEL_FAVORITE        = "favorite"
	EMPTY_NEXT_TIME       = int64(0)
	COUNT_NOT_FOUND       = int64(-1)
)
