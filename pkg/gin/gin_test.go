package gin

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewOptions(t *testing.T) {
	// 创建测试用例
	testCases := []struct {
		name   string
		viper  *viper.Viper
		expect *Options
	}{
		{
			name:   "测试用例1：空的viper",
			viper:  nil,
			expect: &Options{},
		},
		{
			name:   "测试用例2：非空的viper",
			viper:  viper.New(),
			expect: &Options{},
		},
		{
			name: "测试用例3：带有配置的viper",
			viper: func() *viper.Viper {
				v := viper.New()
				v.Set("key", "value")
				return v
			}(),
			expect: &Options{},
		},
	}

	// 运行测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NewOptions(tc.viper)
			assert.Equal(t, tc.expect, result)
		})
	}
}
