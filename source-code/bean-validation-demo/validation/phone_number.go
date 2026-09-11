// Package validation mirrors com.example.beanvalidationdemo.validation: the
// application's two custom constraint annotations and their validators.
package validation

import (
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/beanvalidation"
)

// phoneNumberRegExp is PhoneNumberValidator's expression, unchanged.
//
//	大陆手机号码11位数，匹配格式：前三位固定格式+后8位任意数
//	^ 匹配输入字符串开始的位置
//	\d 匹配一个或多个数字，其中 \ 要转义，所以是 \\d
//	$ 匹配输入字符串结尾的位置
const phoneNumberRegExp = `^[1]((3[0-9])|(4[5-9])|(5[0-3,5-9])|([6][5,6])|(7[0-9])|(8[0-9])|(9[1,8,9]))\d{8}$`

// PhoneNumberMessage is @PhoneNumber's default message.
const PhoneNumberMessage = "Invalid phone number"

// PhoneNumber is the @PhoneNumber constraint annotation, validated by
// IsValidPhoneNumber.
func PhoneNumber() beanvalidation.Constraint {
	return beanvalidation.StringConstraint("PhoneNumber", PhoneNumberMessage, IsValidPhoneNumber)
}

// IsValidPhoneNumber is PhoneNumberValidator.isValid.
//
// A null phone field is valid — the validator says so explicitly ("can be
// null") — so an absent phoneNumber is rejected by @NotNull, never by this.
func IsValidPhoneNumber(phoneField *string) bool {
	if phoneField == nil {
		// can be null
		return true
	}
	return matchesPhoneNumber(*phoneField)
}
