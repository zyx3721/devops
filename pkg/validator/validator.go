package validator

import (
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

var (
	validate *validator.Validate
	trans    ut.Translator
	once     sync.Once
)

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// init 初始化验证器
func init() {
	initValidator()
}

func initValidator() {
	once.Do(func() {
		validate = validator.New()

		// 注册中文翻译
		zhLocale := zh.New()
		uni := ut.New(zhLocale, zhLocale)
		trans, _ = uni.GetTranslator("zh")
		zh_translations.RegisterDefaultTranslations(validate, trans)

		// 使用 json tag 作为字段名
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			// 优先使用 label tag
			if label := fld.Tag.Get("label"); label != "" {
				return label
			}
			// 其次使用 json tag
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" || name == "" {
				return fld.Name
			}
			return name
		})

		// 注册自定义验证规则
		registerCustomValidations()
	})
}

// registerCustomValidations 注册自定义验证规则
func registerCustomValidations() {
	// 注册手机号验证
	validate.RegisterValidation("mobile", func(fl validator.FieldLevel) bool {
		mobile := fl.Field().String()
		if mobile == "" {
			return true // 空值由 required 验证
		}
		// 简单的中国手机号验证：11位，以1开头，第二位是3-9
		if len(mobile) != 11 {
			return false
		}
		if mobile[0] != '1' {
			return false
		}
		// 第二位必须是 3-9
		if mobile[1] < '3' || mobile[1] > '9' {
			return false
		}
		return true
	})

	// 注册自定义翻译
	validate.RegisterTranslation("mobile", trans, func(ut ut.Translator) error {
		return ut.Add("mobile", "{0}必须是有效的手机号", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("mobile", fe.Field())
		return t
	})
}

// Validate 验证结构体
func Validate(obj interface{}) []ValidationError {
	err := validate.Struct(obj)
	if err == nil {
		return nil
	}

	var errors []ValidationError
	for _, e := range err.(validator.ValidationErrors) {
		errors = append(errors, ValidationError{
			Field:   e.Field(),
			Message: e.Translate(trans),
		})
	}
	return errors
}

// ValidateVar 验证单个变量
func ValidateVar(field interface{}, tag string) error {
	return validate.Var(field, tag)
}

// RegisterValidation 注册自定义验证规则
func RegisterValidation(tag string, fn validator.Func) error {
	return validate.RegisterValidation(tag, fn)
}

// RegisterTranslation 注册自定义翻译
func RegisterTranslation(tag string, registerFn validator.RegisterTranslationsFunc, translationFn validator.TranslationFunc) error {
	return validate.RegisterTranslation(tag, trans, registerFn, translationFn)
}

// GetValidator 获取验证器实例
func GetValidator() *validator.Validate {
	return validate
}

// GetTranslator 获取翻译器实例
func GetTranslator() ut.Translator {
	return trans
}

// ValidateAndFormat 验证并格式化错误消息
func ValidateAndFormat(obj interface{}) (bool, string) {
	errors := Validate(obj)
	if len(errors) == 0 {
		return true, ""
	}

	// 返回第一个错误
	return false, errors[0].Message
}

// ValidateAndFormatAll 验证并返回所有错误消息
func ValidateAndFormatAll(obj interface{}) (bool, []string) {
	errors := Validate(obj)
	if len(errors) == 0 {
		return true, nil
	}

	var messages []string
	for _, e := range errors {
		messages = append(messages, e.Message)
	}
	return false, messages
}
