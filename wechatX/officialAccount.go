package wechatX

import (
	"context"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/officialaccount"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
)

// OfficialAccountInstance 公众号操作实例
type OfficialAccountInstance struct {
	Wc              *wechat.Wechat
	OfficialAccount *officialaccount.OfficialAccount
	Cfg             *WechatConfig
}

// NewOfficialAccountInstance new
func NewOfficialAccountInstance(ctx context.Context, cfg *WechatConfig) *OfficialAccountInstance {
	//init config
	wc := wechat.NewWechat()

	redisOpts := &cache.RedisOpts{
		Host:        cfg.RedisConfig.Host,
		Database:    cfg.RedisConfig.Database,
		Password:    cfg.RedisConfig.Password,
		MaxActive:   10,
		MaxIdle:     10,
		IdleTimeout: 60, //second
	}

	redisCache := cache.NewRedis(ctx, redisOpts)
	offCfg := &offConfig.Config{
		AppID:          cfg.AppID,
		AppSecret:      cfg.AppSecret,
		Token:          cfg.Token,
		EncodingAESKey: cfg.EncodingAESKey,
		Cache:          redisCache,
	}
	officialAccount := wc.GetOfficialAccount(offCfg)
	return &OfficialAccountInstance{
		Wc:              wc,
		OfficialAccount: officialAccount,
		Cfg:             cfg,
	}
}
