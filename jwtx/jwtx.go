package jwtx

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang-jwt/jwt/v4/request"
)

// ========================
// JWT Key 常量定义
// ========================

const (
	CtxKeyJwtAccessExpire = "exp"       // 访问令牌的过期时间戳
	CtxKeyJwtIssuedAt     = "iat"       // 令牌签发时间戳
	CtxKeyJwtIssuer       = "iss"       // 签发者标识
	CtxKeyJwtUserId       = "userId"    // 用户 ID
	CtxKeyJwtUserName     = "userName"  // 用户名
	CtxKeyJwtRoleId       = "roleId"    // 角色 ID
	CtxKeyJwtNickName     = "nickName"  // 昵称
	CtxKeyJwtToken        = "tokenStr"  // Token 字符串
	CtxKeyRefreshAt       = "refreshAt" // 可刷新时间点（Unix 秒）
	CtxKeyExpire          = "expire"    // 有效时长（秒）
)

// ========================
// 自定义 Claims 结构
// ========================

// SysJwtClaims 定义了 JWT 的自定义声明结构，包含用户身份信息和标准字段
// 内嵌 jwt.RegisteredClaims 提供标准字段支持（exp、iat、iss 等）
type SysJwtClaims struct {
	UserId               string `json:"userId"`    // 用户 ID
	RoleId               string `json:"roleId"`    // 角色 ID
	DeptId               string `json:"deptId"`    // 部门 ID
	UserName             string `json:"userName"`  // 用户名
	NickName             string `json:"nickName"`  // 昵称
	RefreshAt            int64  `json:"refreshAt"` // 可刷新时间点（Unix 秒）
	Expire               int64  `json:"expire"`    // 令牌有效时长（秒）
	TokenStr             string `json:"tokenStr"`  // 原始 Token 字符串（可选存储）
	jwt.RegisteredClaims        // 标准 JWT Claims：ExpiresAt、IssuedAt、Issuer 等
}

// ========================
// 生成 Token
// ========================

// GetToken 使用 MapClaims 构建 token（适用于动态结构）
func GetToken(secretKey, username, nickName, issuer string, iat, seconds, uid, roleId int64) (string, error) {
	claims := jwt.MapClaims{
		CtxKeyJwtUserId:       uid,
		CtxKeyJwtRoleId:       roleId,
		CtxKeyJwtUserName:     username,
		CtxKeyJwtNickName:     nickName,
		CtxKeyJwtIssuer:       issuer,
		CtxKeyJwtIssuedAt:     iat,
		CtxKeyJwtAccessExpire: iat + seconds,
		CtxKeyRefreshAt:       iat + seconds/2,
		CtxKeyExpire:          seconds,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// GetTokenClaims 使用结构体 SysJwtClaims 构建 token（推荐方式）
func GetTokenClaims(secretKey, username, nickName, issuer string, iat, seconds int64, uid, roleId, deptId string) (string, error) {
	claims := &SysJwtClaims{
		UserId:    uid,
		RoleId:    roleId,
		DeptId:    deptId,
		UserName:  username,
		NickName:  nickName,
		Expire:    seconds,
		RefreshAt: iat + seconds/2,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(time.Unix(iat, 0)),
			ExpiresAt: jwt.NewNumericDate(time.Unix(iat+seconds, 0)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secretKey))
	claims.TokenStr = tokenStr
	return tokenStr, err
}

// MakeJwt 根据自定义 Claims 直接生成 Token
func MakeJwt(secretKey string, claim *SysJwtClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	return token.SignedString([]byte(secretKey))
}

// ========================
// Token 解析函数
// ========================

// ParseToken 解析 Token 并返回自定义 Claims，包含过期检查和 refresh 提示
func ParseToken(tokenString, SigningKey string) (*SysJwtClaims, error) {
	tok, err := stripBearerPrefixFromTokenString(tokenString)
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(tok, &SysJwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SigningKey), nil
	})
	if claims, ok := token.Claims.(*SysJwtClaims); ok {
		if err != nil {
			var ve *jwt.ValidationError
			if errors.As(err, &ve) && ve.Errors&jwt.ValidationErrorExpired != 0 {
				return claims, errors.New("checkRefresh")
			}
			return nil, err
		}
		if token.Valid {
			return claims, nil
		}
	}
	return nil, errors.New("token 无效")
}

// ========================
// 检查 + 自动刷新逻辑
// ========================

// CheckToken 校验 token 有效性，如过期且在可刷新时间内则返回新 token
func CheckToken(tokenStr, secretKey string) (bool, string) {
	claims, err := ParseToken(tokenStr, secretKey)
	if err != nil {
		return false, ""
	}
	now := time.Now().Unix()
	if claims.RefreshAt < now {
		claims.ExpiresAt = jwt.NewNumericDate(time.Unix(now+claims.Expire, 0))
		claims.RefreshAt = now + claims.Expire/2
		claims.IssuedAt = jwt.NewNumericDate(time.Now())
		newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		refreshToken, _ := newToken.SignedString([]byte(secretKey))
		return true, refreshToken
	}
	return true, ""
}

// ========================
// 从上下文中提取字段
// ========================

// GetUserIdFromCtx 从上下文中获取 userId
func GetUserIdFromCtx(ctx context.Context) string {
	if val, ok := ctx.Value(CtxKeyJwtUserId).(string); ok {
		return val
	}
	return ""
}

// GetRoleIdFromCtx 从上下文中获取 roleId
func GetRoleIdFromCtx(ctx context.Context) string {
	if val, ok := ctx.Value(CtxKeyJwtRoleId).(string); ok {
		return val
	}
	return ""
}

// GetTokenStrFromCtx 从上下文中获取 tokenStr
func GetTokenStrFromCtx(ctx context.Context) string {
	if val, ok := ctx.Value(CtxKeyJwtToken).(string); ok {
		return val
	}
	return ""
}

// ========================
// 从 token 中提取字段
// ========================

// GetUserIdFromToken 从 token 中解析 userId
func GetUserIdFromToken(tokenStr, secret string) (string, error) {
	claims, err := ParseToken(tokenStr, secret)
	if err != nil {
		return "", err
	}
	return claims.UserId, nil
}

// GetRoleIdFromToken 从 token 中解析 roleId
func GetRoleIdFromToken(tokenStr, secret string) (string, error) {
	claims, err := ParseToken(tokenStr, secret)
	if err != nil {
		return "", err
	}
	return claims.RoleId, nil
}

// GetDeptIdFromToken 从 token 中解析 deptId
func GetDeptIdFromToken(tokenStr, secret string) (string, error) {
	claims, err := ParseToken(tokenStr, secret)
	if err != nil {
		return "", err
	}
	return claims.DeptId, nil
}

// GetClaimsFromToken 从 token 中解析完整 claims
func GetClaimsFromToken(tokenStr, secret string) (*SysJwtClaims, error) {
	return ParseToken(tokenStr, secret)
}

const JwtPayloadKey = "jwt_payload"

// ParseTokenFromRequest 从 HTTP 请求中解析 JWT，支持 Bearer 自动处理，返回自定义 Claims
func ParseTokenFromRequest(r *http.Request, secret string) (*SysJwtClaims, error) {
	token, err := request.ParseFromRequest(
		r,
		request.AuthorizationHeaderExtractor,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
		request.WithClaims(&SysJwtClaims{}),
	)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*SysJwtClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidId
}

// Strips 'Bearer ' prefix from bearer token string
func stripBearerPrefixFromTokenString(tok string) (string, error) {
	// Should be a bearer token
	if len(tok) > 6 && strings.ToUpper(tok[0:7]) == "BEARER " {
		return tok[7:], nil
	}
	return tok, nil
}
