package cryptx

import (
	"crypto/md5"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/Tang-RoseChild/mahonia"
	"math/rand"
	"strings"
	"time"
)

// RandAuthToken 尝试使用加密随机数生成一个 32 字节的随机字符串并返回其十六进制表示。
// 如果加密随机数生成失败，则退化为使用 RandString(64) 生成一个随机字符串。
// 适用于需要随机令牌或会话 ID 等场景，若对安全性有严格要求，建议在失败后再处理错误。
func RandAuthToken() string {
	buf := make([]byte, 32)
	_, err := crand.Read(buf)
	if err != nil {
		// 如果使用加密随机数失败，则生成一个长度为 64 的普通随机字符串
		return RandString(64)
	}

	// 正常情况下返回 32 字节随机数据的十六进制字符串
	return fmt.Sprintf("%x", buf)
}

// RandString 生成一个长度为 length 的随机字符串。
// 使用 math/rand + time.Now().UnixNano() 作为随机源，并不具备严格的加密安全性。
func RandString(length int64) string {
	sources := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	var result []byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	sourceLength := len(sources)
	var i int64
	for i = 0; i < length; i++ {
		result = append(result, sources[r.Intn(sourceLength)])
	}
	return string(result)
}

// Md5 生成传入字符串的 32 位 MD5 摘要，返回其十六进制表示。
// 仅适用于简单校验、哈希，不适用于加密场景。
func Md5(str string) string {
	m := md5.New()
	m.Write([]byte(str))
	return hex.EncodeToString(m.Sum(nil))
}

// RandNumber 在 [0, max) 范围内生成一个整数随机数。
// 注意使用 math/rand，若需要更强随机性请使用 crypto/rand。
func RandNumber(max int) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Intn(max)
}

// GBK2UTF8 将 GBK 编码字符串转换为 UTF8 编码。
// 返回转换后的字符串及布尔值说明转换是否成功。
func GBK2UTF8(s string) (string, bool) {
	dec := mahonia.NewDecoder("gbk")
	return dec.ConvertStringOK(s)
}

// ConvertEncoding 用于在需要时将 GBK 编码的字符串转换为 UTF8 编码。
// 若转换失败，则原样返回传入的字符串。
func ConvertEncoding(outputGBK string) string {
	// windows 平台的编码可能为 GBK，需要转换为 UTF8 后再进行后续操作
	outputUTF8, ok := GBK2UTF8(outputGBK)
	if ok {
		return outputUTF8
	}
	return outputGBK
}

// ReplaceStrings 依次将数组 old 中的字符串替换为对应 replace 中的字符串。
// 要求 old 与 replace 的长度一致，否则不进行替换。
func ReplaceStrings(s string, old []string, replace []string) string {
	if s == "" {
		return s
	}
	if len(old) != len(replace) {
		return s
	}
	for i, v := range old {
		s = strings.Replace(s, v, replace[i], 1000)
	}
	return s
}

// InStringSlice 判断切片 slice 中是否包含指定元素 element。
// 比较时会去除两边空白。
func InStringSlice(slice []string, element string) bool {
	element = strings.TrimSpace(element)
	for _, v := range slice {
		if strings.TrimSpace(v) == element {
			return true
		}
	}
	return false
}

// EscapeJson 用于转义 JSON 字符串中的特殊字符，例如换行、制表符等。
func EscapeJson(s string) string {
	specialChars := []string{"\\", "\b", "\f", "\n", "\r", "\t", "\""}
	replaceChars := []string{"\\\\", "\\b", "\\f", "\\n", "\\r", "\\t", "\\\""}
	return ReplaceStrings(s, specialChars, replaceChars)
}
