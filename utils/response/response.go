package response

import (
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zhanghaidi/zero-common/utils/errorx"

	"net/http"
)

type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"` // 仅当有数据时才包含 Data 字段
}

// Response 处理 HTTP 响应，支持 ApiError 进行错误处理
func Response(w http.ResponseWriter, resp interface{}, err error) {
	var body Body

	if err != nil {
		logx.Error(err)
		// 检查是否为 CodeError 类型
		var codeErr *errorx.CodeError
		if errors.As(err, &codeErr) {
			body.Code = codeErr.Code
			body.Msg = codeErr.Msg
		} else {
			// 处理非 CodeError 类型的错误
			body.Code = errorx.DefaultCode
			body.Msg = err.Error()
		}
	} else {
		// 成功响应
		body.Code = 0
		body.Msg = "ok"
		body.Data = resp
	}

	httpx.OkJson(w, body)
}

// CodeResponse 提供自定义响应方法
func CodeResponse(code int, msg string, data interface{}) Body {
	return Body{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}
