package svc

import "errors"

// ErrNoFeedsEnabled 错误：没有启用任何价格源
var ErrNoFeedsEnabled = errors.New("no exchange feeds enabled")

// ErrStorageInitFailed 错误：存储初始化失败
var ErrStorageInitFailed = errors.New("storage initialization failed")
