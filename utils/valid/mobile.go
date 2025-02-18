package valid

import (
	"github.com/zhanghaidi/zero-common/utils/errorx"
	"regexp"
)

func VerifyMobile(mobile string) error {
	if len(mobile) != 11 {
		return errorx.NewCodeError(1, "手机号位数不对")
	}

	reg := `^[1]([1-9])[0-9]{9}$`
	rgx := regexp.MustCompile(reg)
	if !rgx.MatchString(mobile) {
		return errorx.NewCodeError(1, "手机号格式不正确")
	}
	return nil
}
