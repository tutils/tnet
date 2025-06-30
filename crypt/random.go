package crypt

import "math/rand"

// 验证接口实现
var _ rand.Source = (*LCGSource)(nil)
var _ rand.Source64 = (*LCGSource)(nil)

// LCGSource 结构体存储伪随机状态
type LCGSource struct {
	state uint64
}

// 常用64位LCG参数（Numerical Recipes）
const (
	a = 6364136223846793005
	c = 1442695040888963407
)

func NewLCGSource(seed int64) rand.Source {
	return &LCGSource{state: uint64(seed)}
}

// Seed 实现rand.Source接口
func (l *LCGSource) Seed(seed int64) {
	l.state = uint64(seed)
}

// Uint64 实现rand.Source64接口
func (l *LCGSource) Uint64() uint64 {
	l.state = l.state*a + c
	return l.state
}

// Int63 生成63位随机数
func (l *LCGSource) Int63() int64 {
	return int64(l.Uint64() >> 1) // 右移确保63位正整数
}
