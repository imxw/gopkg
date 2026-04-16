// Package validatorx provides validation utilities with Chinese error messages.
//
// WARNING: This package uses init() to replace Gin's default validator.
// Importing this package will globally modify gin/binding.Validator.
// For non-Gin projects, call GetValidator() directly instead.
package validatorx

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTrans "github.com/go-playground/validator/v10/translations/zh"
)

// -------------------------- 错误定义 --------------------------
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag,omitempty"`
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	switch len(ve) {
	case 0:
		return ""
	case 1:
		return ve[0].Message
	default:
		msgs := make([]string, 0, len(ve))
		for _, err := range ve {
			msgs = append(msgs, err.Message)
		}
		return strings.Join(msgs, "; ")
	}
}

var ErrNilData = errors.New("验证数据不能为空")

// -------------------------- 验证器核心 --------------------------
type Validator struct {
	validate *validator.Validate
	trans    ut.Translator
}

// -------------------------- 全局实例 --------------------------
var (
	once             sync.Once
	defaultValidator *Validator
)

// GetValidator 获取验证器（单例）
func GetValidator() *Validator {
	once.Do(func() {
		defaultValidator = newValidator()
	})
	return defaultValidator
}

// -------------------------- Gin 验证器适配器 --------------------------
// ginValidatorAdapter 将 *validator.Validate 适配为 Gin 的 binding.StructValidator
type ginValidatorAdapter struct {
	v     *validator.Validate
	trans ut.Translator
}

func (a *ginValidatorAdapter) ValidateStruct(obj any) error {
	err := a.v.Struct(obj)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		return &translatedError{verrs: verrs, trans: a.trans}
	}
	return err
}

func (a *ginValidatorAdapter) Engine() interface{} {
	return a.v
}

// translatedError 包装验证错误，Error() 返回中文消息
type translatedError struct {
	verrs validator.ValidationErrors
	trans ut.Translator
}

func (e *translatedError) Error() string {
	return parseValidationErrors(e.verrs, e.trans).Error()
}

// ReplaceGinValidator 用自定义验证器替换 Gin 默认验证器
func ReplaceGinValidator() {
	v := GetValidator()
	binding.Validator = &ginValidatorAdapter{v: v.validate, trans: v.trans}
}

// -------------------------- 验证器创建 --------------------------
func newValidator() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.SetTagName("binding")

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		if fld.Anonymous {
			return ""
		}
		if fld.Tag.Get("binding") == "-" {
			return ""
		}
		name := getFieldName(fld)
		return strings.ToLower(name[:1]) + name[1:]
	})

	trans := initTranslator(v)

	return &Validator{
		validate: v,
		trans:    trans,
	}
}

func getFieldName(fld reflect.StructField) string {
	for _, tag := range []string{"form", "json"} {
		if name := fld.Tag.Get(tag); name != "" && name != "-" {
			return strings.SplitN(name, ",", 2)[0]
		}
	}
	return fld.Name
}

func initTranslator(v *validator.Validate) ut.Translator {
	zhLocale := zh.New()
	uni := ut.New(zhLocale, zhLocale)
	trans, _ := uni.GetTranslator("zh")

	if trans != nil {
		_ = zhTrans.RegisterDefaultTranslations(v, trans)
		registerTagTranslations(v, trans)
	}
	return trans
}

func registerTagTranslations(v *validator.Validate, trans ut.Translator) {
	if trans == nil {
		return
	}

	tagTrans := map[string]struct {
		msg       string
		transFunc func(ut ut.Translator, fe validator.FieldError) string
	}{
		"required": {
			msg: "{0}为必填字段",
			transFunc: func(ut ut.Translator, fe validator.FieldError) string {
				msg, _ := ut.T("required", fe.Field())
				return msg
			},
		},
		"gte": {
			msg: "{0}必须大于等于{1}",
			transFunc: func(ut ut.Translator, fe validator.FieldError) string {
				msg, _ := ut.T("gte", fe.Field(), fe.Param())
				return msg
			},
		},
		"lte": {
			msg: "{0}必须小于等于{1}",
			transFunc: func(ut ut.Translator, fe validator.FieldError) string {
				msg, _ := ut.T("lte", fe.Field(), fe.Param())
				return msg
			},
		},
	}

	for tag, item := range tagTrans {
		_ = v.RegisterTranslation(tag, trans,
			func(ut ut.Translator) error {
				return ut.Add(tag, item.msg, true)
			},
			item.transFunc,
		)
	}
}

// -------------------------- 核心验证方法 --------------------------
func (v *Validator) Validate(data any) ValidationErrors {
	if data == nil || (reflect.ValueOf(data).Kind() == reflect.Ptr && reflect.ValueOf(data).IsNil()) {
		return ValidationErrors{{Field: "", Message: ErrNilData.Error()}}
	}

	if err := v.validate.Struct(data); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			return parseValidationErrors(verrs, v.trans)
		}
		return ValidationErrors{{Field: "", Message: fmt.Sprintf("参数验证失败: %v", err)}}
	}

	return nil
}

func parseValidationErrors(verrs validator.ValidationErrors, trans ut.Translator) ValidationErrors {
	errs := make(ValidationErrors, 0, len(verrs))

	for _, e := range verrs {
		message := e.Translate(trans)

		cleanField := e.Field()
		if strings.Contains(cleanField, ".") {
			parts := strings.Split(cleanField, ".")
			cleanField = parts[len(parts)-1]
		}
		if cleanField != "" {
			cleanField = strings.ToLower(cleanField[:1]) + cleanField[1:]
		}

		errs = append(errs, ValidationError{
			Field:   cleanField,
			Message: message,
			Tag:     e.Tag(),
		})
	}
	return errs
}

// 包初始化：自动替换Gin验证器
func init() {
	ReplaceGinValidator()
}
