package model

type Response struct {
	Code int `json:"code" example:"200"` //状态码，和http状态码剥离，以分离网络和业务
	Msg  any `json:"msg"`                //信息。可以是NULL，也可以是成功时返回的结构体，也可以是失败时的错误信息
}
