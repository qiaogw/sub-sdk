package wechatX

// RedisConfig 微信 config
type RedisConfig struct {
	Host        string `yaml:"host"`
	Password    string `yaml:"password"`
	Database    int    `yaml:"database"`
	MaxActive   int    `yaml:"maxActive"`
	MaxIdle     int    `yaml:"maxIdle"`
	IdleTimeout int    `yaml:"idleTimeout"`
}

// WechatConfig 公众号相关配置
type WechatConfig struct {
	AppID          string `yaml:"appID"`
	AppSecret      string `yaml:"appSecret"`
	Token          string `yaml:"token"`
	EncodingAESKey string `yaml:"encodingAESKey"`
	RedisConfig    RedisConfig
}
