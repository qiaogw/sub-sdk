package wechatX

import (
	"context"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/openplatform"
	"github.com/silenceper/wechat/v2/openplatform/config"
)

// OpenplatformAccountInstance 公众号操作样例
type OpenplatformAccountInstance struct {
	Wc                  *wechat.Wechat
	OpenplatformAccount *openplatform.OpenPlatform
	Cfg                 *WechatConfig
}

// InitWechat 获取wechat实例
// 在这里已经设置了全局cache，则在具体获取公众号/小程序等操作实例之后无需再设置，设置即覆盖
func InitWechat() *wechat.Wechat {
	wc := wechat.NewWechat()
	//wc.SetCache(rc)
	return wc
}

// NewOpenplatformAccount new
func NewOpenplatformAccount(ctx context.Context, cfg *WechatConfig) *OpenplatformAccountInstance {
	//init config
	redisOpts := &cache.RedisOpts{
		Host:        cfg.RedisConfig.Host,
		Password:    cfg.RedisConfig.Password,
		Database:    cfg.RedisConfig.Database,
		MaxActive:   cfg.RedisConfig.MaxActive,
		MaxIdle:     cfg.RedisConfig.MaxIdle,
		IdleTimeout: cfg.RedisConfig.IdleTimeout,
	}
	redisCache := cache.NewRedis(ctx, redisOpts)
	wc := wechat.NewWechat()
	offCfg := &config.Config{
		AppID:          cfg.AppID,
		AppSecret:      cfg.AppSecret,
		Token:          cfg.Token,
		EncodingAESKey: cfg.EncodingAESKey,
		Cache:          redisCache,
	}
	openplatformAccount := wc.GetOpenPlatform(offCfg)
	return &OpenplatformAccountInstance{
		Wc:                  wc,
		OpenplatformAccount: openplatformAccount,
	}
}
