package errmsg

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CodeError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *CodeError) Error() string {
	return fmt.Sprintf("ErrCode:%d, Errmsg:%s", e.Code, e.Msg)
}

func NewErrCode(code int) error {
	return &CodeError{
		Code: code,
		Msg:  GetErrMsg(code),
	}
}

func NewErrCodeMsg(code int, msg string) error {
	return &CodeError{
		Code: code,
		Msg:  msg,
	}
}

func NewGrpcErr(code int, msg string) error {
	return status.Error(codes.Code(code), msg)
}

const (
	Success             = 200
	Error               = 500
	CodeServerBusy      = 1015
	ErrorServerCommon   = 5001
	ErrorDbUpdate       = 5002
	ErrorDbSelect       = 5003
	ErrorDbInsert       = 5004 // 数据库插入失败
	ErrorDbTransaction  = 5005 // 事务操作失败
	ErrorSnowflakeID    = 5006 // 雪花ID生成失败
	ErrorJsonMarshal    = 5007 // JSON序列化失败
	ErrorJsonUnmarshal  = 5008 // JSON反序列化失败
	ErrorKafkaPush      = 5009 // Kafka消息发送失败
	ErrorDelayMsg       = 5010 // 延时消息发送失败
	ErrorRedisSelect    = 5011
	ErrorRedisUpdate    = 5012
	ErrorUserExist      = 1001
	ErrorLoginWrong     = 1002
	ErrorUserNotExist   = 1003
	ErrorTokenNotExist  = 1004
	ErrorTokenTypeWrong = 1005
	ErrorTokenRuntime   = 1006
	ErrorTokenRefresh   = 1007
	ErrorUserNoRight    = 1008
	ErrorUserNoLogin    = 1009
	ErrorUserLogined    = 1010
	ErrorUserBanned     = 1011
	ErrorInputWrong     = 2001
	ErrorTypeTransfer   = 2002
	ErrorBuildOutbox    = 3001
)

var codeMsg = map[int]string{
	Success:             "OK",
	Error:               "FAIL",
	CodeServerBusy:      "服务繁忙",
	ErrorServerCommon:   "系统内部错误",
	ErrorDbUpdate:       "更新数据库失败",
	ErrorDbSelect:       "查询数据库失败",
	ErrorDbInsert:       "数据库插入失败",
	ErrorDbTransaction:  "事务操作失败",
	ErrorSnowflakeID:    "雪花ID生成失败",
	ErrorJsonMarshal:    "JSON序列化失败",
	ErrorJsonUnmarshal:  "JSON反序列化失败",
	ErrorKafkaPush:      "Kafka消息发送失败",
	ErrorDelayMsg:       "延时消息发送失败",
	ErrorRedisSelect:    "Redis查询失败",
	ErrorRedisUpdate:    "Redis更新失败",
	ErrorUserExist:      "用户名已存在",
	ErrorLoginWrong:     "用户名或密码错误",
	ErrorUserNotExist:   "用户不存在",
	ErrorTokenNotExist:  "TOKEN不存在",
	ErrorTokenTypeWrong: "TOKEN格式错误",
	ErrorTokenRuntime:   "TOKEN已过期",
	ErrorTokenRefresh:   "TOKEN刷新失败",
	ErrorUserNoRight:    "权限不足",
	ErrorUserNoLogin:    "未登录",
	ErrorUserLogined:    "已登录",
	ErrorUserBanned:     "用户已被封禁",
	ErrorInputWrong:     "输入有误",
	ErrorTypeTransfer:   "类型转换错误",
	ErrorBuildOutbox:    "构建 Outbox 消息载荷失败",
}

func GetErrMsg(code int) string {
	msg, ok := codeMsg[code]
	if !ok {
		return codeMsg[Error]
	}
	return msg
}
