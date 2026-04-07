package password

import (
	"crypto/rand"
	"math/big"
)

const defaultLen = 16

var (
	lower   = "abcdefghijklmnopqrstuvwxyz"
	upper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits  = "0123456789"
	symbols = "!@#$%^&*()-_=+[]{}<>?/|~"
	all     = lower + upper + digits + symbols
)

func secureInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func shuffleBytes(b []byte) {
	for i := len(b) - 1; i > 0; i-- {
		j := secureInt(i + 1)
		b[i], b[j] = b[j], b[i]
	}
}

// Generate 生成一个随机强密码，默认长度 16，确保包含大小写字母、数字和符号
func Generate() string {
	n := defaultLen
	b := make([]byte, 0, n)

	// 保证每类字符至少一个
	b = append(b, lower[secureInt(len(lower))])
	b = append(b, upper[secureInt(len(upper))])
	b = append(b, digits[secureInt(len(digits))])
	b = append(b, symbols[secureInt(len(symbols))])

	for len(b) < n {
		b = append(b, all[secureInt(len(all))])
	}

	shuffleBytes(b)
	return string(b)
}

// BatchGenerate 批量生成指定数量的随机强密码
func BatchGenerate(count int) []string {
	passwords := make([]string, count)
	for i := 0; i < count; i++ {
		passwords[i] = Generate()
	}
	return passwords
}
