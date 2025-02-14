package wechatX

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/chanxuehong/wechat/mp/core"
	mpoauth2 "github.com/chanxuehong/wechat/mp/oauth2"
	"github.com/chanxuehong/wechat/oauth2"
	"net/http"
	"sort"
	"strings"
)

func GetWechatOauth2Client(appId string, appSecret string) *oauth2.Client {
	oauth2Endpoint := mpoauth2.NewEndpoint(appId, appSecret)
	oauth2Client := oauth2.Client{
		Endpoint: oauth2Endpoint,
	}
	return &oauth2Client
}

func GetWechatClient(appId string, appSecret string, httpClient *http.Client) *core.Client {
	accessTokenServer := core.NewDefaultAccessTokenServer(appId, appSecret, httpClient)
	wechatClient := core.NewClient(accessTokenServer, nil)
	return wechatClient
}

/**
 * 获取 微信登陆的url
 */

func GetWxLoginUrl(appid, fillwebsite string) string {
	return fmt.Sprintf(`https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo`, appid, fillwebsite)
}

const Token = "sub-token"

func ValidateSignature(timestamp, nonce, signature string) bool {
	params := []string{Token, timestamp, nonce}
	sort.Strings(params)
	str := strings.Join(params, "")
	hash := sha1.New()
	hash.Write([]byte(str))
	encrypted := hex.EncodeToString(hash.Sum(nil))
	return encrypted == signature
}
