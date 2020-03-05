package common

import "errors"

var (
	ERR_LOCK_ALREDY_REQUIRED = errors.New("锁已被占用")
)
