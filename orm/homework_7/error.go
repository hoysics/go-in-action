package homework_7

import (
	"github.com/hoysics/go-in-action/orm/homework_7/internal/errs"
)

// 将内部的 sentinel error 暴露出去
var (
	// ErrNoRows 代表没有找到数据
	ErrNoRows = errs.ErrNoRows
)
