package captchax

import (
	"github.com/google/uuid"
	"github.com/mojocn/base64Captcha"
	"image/color"
)

// SetStore 设置验证码存储器，将传入的 store 赋值给默认的内存存储器
func SetStore(s base64Captcha.Store) {
	base64Captcha.DefaultMemStore = s
}

// configJsonBody 表示 JSON 请求体的结构，用于生成验证码时配置相关参数
type configJsonBody struct {
	Id            string                       // 验证码 ID
	CaptchaType   string                       // 验证码类型
	VerifyValue   string                       // 验证值
	DriverAudio   *base64Captcha.DriverAudio   // 音频验证码驱动
	DriverString  *base64Captcha.DriverString  // 字符串验证码驱动
	DriverChinese *base64Captcha.DriverChinese // 中文验证码驱动
	DriverMath    *base64Captcha.DriverMath    // 数学验证码驱动
	DriverDigit   *base64Captcha.DriverDigit   // 数字验证码驱动
}

// DriverStringFunc 生成基于字符串的验证码
// 返回值包括验证码 ID、Base64 编码的图片字符串以及可能出现的错误
func DriverStringFunc(length int) (id, b64s string, err error) {
	// 初始化配置结构体，并生成一个唯一 ID
	e := configJsonBody{}
	e.Id = uuid.New().String()
	// 创建字符串验证码驱动，设置验证码尺寸、干扰项、字符集等参数
	e.DriverString = base64Captcha.NewDriverString(
		46, 140, 2, 2, length, "234567890-captcha-captcha",
		&color.RGBA{R: 240, G: 240, B: 246, A: 246}, nil,
		[]string{"captcha-haha-mimi"})
	// 调用 ConvertFonts 转换字体（若有自定义字体可进行处理）
	driver := e.DriverString.ConvertFonts()
	// 创建验证码实例，并生成验证码
	capt := base64Captcha.NewCaptcha(driver, base64Captcha.DefaultMemStore)
	id, b64s, _, err = capt.Generate()
	return id, b64s, err
}

// DriverDigitFunc 生成基于数字的验证码
// 返回值包括验证码 ID、Base64 编码的图片字符串以及可能出现的错误
func DriverDigitFunc(length int) (id, b64s string, err error) {
	// 初始化配置结构体，并生成一个唯一 ID
	e := configJsonBody{}
	e.Id = uuid.New().String()
	// 创建数字验证码驱动，设置验证码高度、宽度、位数、干扰程度等参数
	e.DriverDigit = base64Captcha.NewDriverDigit(80, 240, length, 0.7, 80)
	driver := e.DriverDigit
	// 创建验证码实例，并生成验证码
	capt := base64Captcha.NewCaptcha(driver, base64Captcha.DefaultMemStore)
	//return cap.Generate()
	id, b64s, _, err = capt.Generate()
	return id, b64s, err
}

// Verify 校验验证码
// 参数 id 为验证码 ID，code 为用户输入的验证码字符串，clear 指定是否校验后清除验证码
// 返回 true 表示验证码验证通过，false 表示验证失败
func Verify(id, code string, clear bool) bool {
	return base64Captcha.DefaultMemStore.Verify(id, code, clear)
}
