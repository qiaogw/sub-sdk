package rpcmiddle

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qiaogw/sub-sdk/errx"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggerInterceptor rpc服务日志拦截中间件
// 自定义错误只给前端该错误代码定义的简短错误信息
// 详细错误信息打印到日志
func LoggerInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	resp, err = handler(ctx, req)
	if err != nil {
		logx.WithContext(ctx).Errorf("❌【RPC-SRV-ERR】 %+v", err)
		causeErr := errors.Cause(err)                // err类型
		if e, ok := causeErr.(*errx.CodeError); ok { //自定义错误类型
			//自定义错误只给前端该错误代码定义的简短错误信息
			err = status.Error(codes.Code(e.GetErrCode()), e.GetErrMsg())
		}
	}
	return resp, err
}
