package validatorx

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestReq 测试用结构体
type TestReq struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required,min=6"`
	Nickname string `json:"nickname" form:"nickname"`
}

// NestedReq 嵌套结构体
type NestedReq struct {
	User  TestReq  `json:"user" form:"user" binding:"required"`
	Tags  []string `json:"tags" form:"tags"`
	Email *string  `json:"email" form:"email"`
}

// TestValidator_Validate 测试结构体验证核心逻辑
func TestValidator_Validate(t *testing.T) {
	tests := []struct {
		name    string
		data    any
		wantErr bool
		errMsg  string
	}{
		{
			name:    "空数据验证",
			data:    nil,
			wantErr: true,
			errMsg:  ErrNilData.Error(),
		},
		{
			name: "必填字段为空",
			data: &TestReq{
				Username: "",
				Password: "123456",
			},
			wantErr: true,
			errMsg:  "username",
		},
		{
			name: "密码长度不足",
			data: &TestReq{
				Username: "test",
				Password: "12345",
			},
			wantErr: true,
			errMsg:  "password",
		},
		{
			name: "验证通过",
			data: &TestReq{
				Username: "test",
				Password: "123456",
			},
			wantErr: false,
		},
	}

	validator := GetValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.Validate(tt.data)
			if tt.wantErr {
				assert.NotNil(t, errs)
				assert.NotEmpty(t, errs.Error())
				if tt.errMsg != "" {
					assert.Contains(t, errs.Error(), tt.errMsg)
				}
			} else {
				assert.Nil(t, errs)
			}
		})
	}
}

// TestValidator_EmptyOptionalFields 测试可选字段为空的情况
func TestValidator_EmptyOptionalFields(t *testing.T) {
	validator := GetValidator()

	req := &TestReq{
		Username: "test",
		Password: "123456",
		Nickname: "",
	}

	errs := validator.Validate(req)
	assert.Nil(t, errs)
}

// TestValidator_ErrorFormat 测试错误格式
func TestValidator_ErrorFormat(t *testing.T) {
	validator := GetValidator()

	req := &TestReq{
		Username: "",
		Password: "123",
	}

	errs := validator.Validate(req)
	assert.NotNil(t, errs)

	errStr := errs.Error()
	assert.NotEmpty(t, errStr)
	assert.Contains(t, errStr, "username")

	jsonData, err := json.Marshal(errs)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	assert.True(t, len(errs) >= 1)
	assert.NotEmpty(t, errs[0].Field)
	assert.NotEmpty(t, errs[0].Message)
}

// TestValidator_NilPointer 测试空指针处理
func TestValidator_NilPointer(t *testing.T) {
	validator := GetValidator()

	errs := validator.Validate(nil)
	assert.NotNil(t, errs)
	assert.Contains(t, errs.Error(), ErrNilData.Error())

	var req *TestReq
	errs = validator.Validate(req)
	assert.NotNil(t, errs)
	assert.Contains(t, errs.Error(), ErrNilData.Error())
}

// TestValidator_NestedStruct 测试嵌套结构体验证
func TestValidator_NestedStruct(t *testing.T) {
	validator := GetValidator()

	t.Run("嵌套结构体验证失败", func(t *testing.T) {
		req := &NestedReq{
			User: TestReq{
				Username: "",
				Password: "123456",
			},
		}

		errs := validator.Validate(req)
		assert.NotNil(t, errs)
		assert.Contains(t, errs.Error(), "username")
	})

	t.Run("嵌套结构体验证成功", func(t *testing.T) {
		req := &NestedReq{
			User: TestReq{
				Username: "test",
				Password: "123456",
			},
		}

		errs := validator.Validate(req)
		assert.Nil(t, errs)
	})
}

// TestGetValidator_Singleton 测试单例模式
func TestGetValidator_Singleton(t *testing.T) {
	v1 := GetValidator()
	v2 := GetValidator()

	assert.NotNil(t, v1)
	assert.NotNil(t, v2)
	assert.Same(t, v1, v2)
}

// TestValidationErrors_Error 测试错误信息格式化
func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   ValidationErrors
		expected string
	}{
		{
			name:     "空错误列表",
			errors:   ValidationErrors{},
			expected: "",
		},
		{
			name: "单个错误",
			errors: ValidationErrors{
				{Field: "username", Message: "用户名为必填字段"},
			},
			expected: "用户名为必填字段",
		},
		{
			name: "多个错误",
			errors: ValidationErrors{
				{Field: "username", Message: "用户名为必填字段"},
				{Field: "password", Message: "密码长度不足"},
			},
			expected: "用户名为必填字段; 密码长度不足",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.errors.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}
