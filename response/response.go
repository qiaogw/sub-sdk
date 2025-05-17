package response

import (
	"github.com/pkg/errors"
	"github.com/qiaogw/sub-sdk/errx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/status"
	"net/http"
	"reflect"
	"strings"
)

type Body struct {
	Code uint32      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// Response 统一封装 API 响应
// 自定义错误只给前端该错误代码定义的简短错误信息
// 若错误代码为前端请求错误（errx.ErrReq），返回详细信息
// 详细错误信息打印到日志
func Response(r *http.Request, w http.ResponseWriter, resp interface{}, err error) {
	body := Body{}

	if err != nil {
		body.Code, body.Msg = parseError(err)
		logx.WithContext(r.Context()).Errorf("【API-ERR】 : %+v ", err)
	} else {
		body.Msg = "请求成功!"
		body.Data = resp
		rt := reflect.TypeOf(resp)
		if rt.String() == "*types.CommonResponse" {
			rv := reflect.ValueOf(resp).Elem()
			if rv.Kind() == reflect.Struct {
				dataField := rv.FieldByName("Data")
				msgField := rv.FieldByName("Msg")
				body.Msg = getStringIfValid(msgField, "请求成功!")
				body.Data = getValueIfValid(dataField)
			}
		}
		logx.WithContext(r.Context()).Debugf("【API-OK】")
	}

	httpx.OkJson(w, body)
}

// 解析错误，返回错误码和消息
func parseError(err error) (uint32, string) {
	errcode := errx.ServerCommonError
	errmsg := "服务器开小差啦，稍后再来试一试"

	causeErr := errors.Cause(err)
	if e, ok := causeErr.(*errx.CodeError); ok {
		errcode = e.GetErrCode()
		if errcode == errx.ErrReq {
			return e.GetErrCode(), e.GetErrMsg()
		}
		return errcode, errx.MapErrMsg(errcode)
	}

	if gstatus, ok := status.FromError(causeErr); ok {
		grpcCode := uint32(gstatus.Code())
		if errx.IsCodeErr(grpcCode) {
			switch grpcCode {
			case errx.NoData, errx.Success, errx.ErrReq:
				return grpcCode, gstatus.Message()
			default:
				errmsg = errx.MapErrMsg(grpcCode)
			}
			// 检测主键重复错误
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return errx.Duplicate, errx.MapErrMsg(errx.Duplicate)
			}
		}
	} else {
		errmsg = err.Error()
	}

	return errcode, errmsg
}

// 判断是否为 nil
func isNil(i interface{}) bool {
	defer func() { recover() }()
	vi := reflect.ValueOf(i)
	return vi.Kind() == reflect.Ptr && vi.IsNil()
}

// 获取字段值（如果有效）
func getValueIfValid(v reflect.Value) interface{} {
	if v.IsValid() {
		return v.Interface()
	}
	return nil
}

// 获取字符串字段值（如果有效）
func getStringIfValid(v reflect.Value, defaultVal string) string {
	if v.IsValid() && v.Kind() == reflect.String {
		return v.String()
	}
	return defaultVal
}
