package response

import (
	"errors"
	"fmt"
	errp "github.com/pkg/errors"
	"github.com/qiaogw/sub-sdk/errx"
	"github.com/zeromicro/go-zero/core/logx"
	errz "github.com/zeromicro/x/errors"
	xhttp "github.com/zeromicro/x/http"
	"google.golang.org/grpc/status"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

type Body struct {
	Code uint32      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}
type FileResponse struct {
	Filename string
	Mime     string // 可选
	Content  []byte
	Inline   bool // 是否浏览器预览而非下载
}

// Response 统一封装 API 响应
// 自定义错误（errx.ErrReq）只给前端该错误代码定义的简短错误信息
// 详细错误信息打印到日志
func Response(r *http.Request, w http.ResponseWriter, resp any, err error) {
	body := Body{}
	if err != nil {
		// 错误统一处理
		body.Code, body.Msg = parseError(err)
		logx.WithContext(r.Context()).Errorf("❌【API-ERR】: %+v", err)
		er := errz.New(int(body.Code), body.Msg)
		xhttp.JsonBaseResponseCtx(r.Context(), w, er)
		return
	}

	// ✅ 响应为 FileResponse（如导出 Excel / PDF / 图片等）
	if f, ok := resp.(*FileResponse); ok {
		fmime := f.Mime
		if fmime == "" {
			fmime = mime.TypeByExtension(filepath.Ext(f.Filename)) // 自动推断
			if fmime == "" {
				fmime = "application/octet-stream"
			}
		}
		dispositionType := "attachment"
		if f.Inline {
			dispositionType = "inline"
		}

		w.Header().Set("Content-Type", fmime)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, dispositionType, f.Filename))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		_, _ = w.Write(f.Content)
		return
	}

	// ✅ 正常 JSON 返回（结构体、map、Pydantic 类等）
	xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
}

// 解析错误，返回错误码和消息
func parseError(err error) (uint32, string) {
	errcode := errx.ServerCommonError
	errmsg := "服务器开小差啦，稍后再来试一试"
	causeErr := errp.Cause(err)
	var e *errx.CodeError
	if errors.As(causeErr, &e) {
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
		}
	} else {
		errmsg = err.Error()
	}
	// 检测主键重复错误
	if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return errx.Duplicate, errx.MapErrMsg(errx.Duplicate)
	}
	return errcode, errmsg
}
