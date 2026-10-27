package validatorx

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin/binding"
)

// 覆盖缺口补测：gin 适配器（ValidateStruct/Engine）、ReplaceGinValidator、
// 标签名解析三态（json tag / 无 tag 回退字段名 / binding:"-" 忽略）。

// TagProbe 标签名解析三态探针。
type TagProbe struct {
	WithJSON   string `json:"withJson" form:"withJson" binding:"required"`
	NoTag      string `binding:"required"`
	Skipped    string `binding:"-"`
	MultiParts string `json:"multiPart,inline" binding:"required"`
}

func TestValidator_TagNameResolution(t *testing.T) {
	errs := GetValidator().Validate(TagProbe{})
	joined := errs.Error()

	// json tag 首字母小写化 + 逗号截断。
	found := false
	for _, e := range errs {
		if e.Field == "withJson" {
			found = true
		}
	}
	if !found {
		t.Fatalf("json tag 字段应解析为 withJson，got %q", joined)
	}
	// 无 tag 回退字段名（首字母小写）。
	found = false
	for _, e := range errs {
		if e.Field == "noTag" {
			found = true
		}
	}
	if !found {
		t.Fatalf("无 tag 字段应回退字段名 noTag，got %q", joined)
	}
	// binding:"-" 不参与验证（无其错误项）。
	for _, e := range errs {
		if e.Field == "skipped" {
			t.Fatal(`binding:"-" 字段不应参与验证`)
		}
	}
	if !strings.Contains(joined, "multiPart") {
		t.Fatalf("json tag 逗号截断取首段，got %q", joined)
	}
}

func TestGinValidatorAdapter_Struct(t *testing.T) {
	ReplaceGinValidator()
	adapter, ok := binding.Validator.(interface {
		ValidateStruct(any) error
		Engine() any
	})
	if !ok {
		t.Fatal("binding.Validator should be the ginValidatorAdapter after Replace")
	}

	// 合法数据：nil。
	if err := adapter.ValidateStruct(TestReq{Username: "u", Password: "123456"}); err != nil {
		t.Fatalf("valid struct should pass, got %v", err)
	}
	// 非法数据：错误链非 nil 且 Error() 是中文翻译产物。
	err := adapter.ValidateStruct(TestReq{Username: "", Password: "1"})
	if err == nil {
		t.Fatal("invalid struct should fail")
	}
	if msg := err.Error(); msg == "" || strings.Contains(msg, "Key:") {
		t.Fatalf("error should be translated Chinese message, got %q", msg)
	}
	// 非验证错误（nil / 非法输入形态）原样透传（validator 库对 nil/非
	// 结构体返回自身错误，适配器不翻译——只翻译 ValidationErrors）。
	if err := adapter.ValidateStruct(nil); err == nil {
		t.Fatal("nil data surfaces the underlying validator error (pass-through)")
	}
	if err := adapter.ValidateStruct("not-a-struct"); err == nil {
		t.Fatal("non-struct input should surface validator error")
	}
	// Engine 暴露底层 *validator.Validate。
	if adapter.Engine() == nil {
		t.Fatal("engine should expose the underlying validator")
	}
}

// 多错误聚合：ValidationErrors.Error() 的分号连接腿（单错误腿已有测试）。
func TestValidationErrors_MultipleJoined(t *testing.T) {
	errs := GetValidator().Validate(TestReq{})
	if len(errs) < 2 {
		t.Fatalf("two required fields should yield >=2 errors, got %d", len(errs))
	}
	msg := errs.Error()
	if !strings.Contains(msg, "; ") {
		t.Fatalf("multiple errors should join with '; ', got %q", msg)
	}
	if msg := (ValidationErrors{}).Error(); msg != "" {
		t.Fatal("empty ValidationErrors should render empty string")
	}
}

// gte/lte 翻译分支：数值越界的中文消息（覆盖 registerTagTranslations 的
// 两个带参 tag——required 腿已由既有用例覆盖）。
type RangeReq struct {
	Age     int `json:"age" binding:"gte=18,lte=65"`
	Workers int `json:"workers" binding:"gte=1"`
}

func TestValidator_RangeTags_Translated(t *testing.T) {
	errs := GetValidator().Validate(RangeReq{Age: 10, Workers: 0})
	if len(errs) != 2 {
		t.Fatalf("want 2 range violations, got %d: %v", len(errs), errs)
	}
	byField := map[string]string{}
	for _, e := range errs {
		byField[e.Field] = e.Message
	}
	if m, ok := byField["age"]; !ok || !strings.Contains(m, "大于等于") || !strings.Contains(m, "18") {
		t.Fatalf("age violation should be translated gte message with param, got %q", m)
	}
	if m, ok := byField["workers"]; !ok || !strings.Contains(m, "大于等于") {
		t.Fatalf("workers violation should be translated, got %q", m)
	}
	// lte 腿：越上限。
	errs = GetValidator().Validate(RangeReq{Age: 70, Workers: 1})
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "小于等于") {
		t.Fatalf("age over max should be translated lte message, got %v", errs)
	}
	// 区间内：无错误。
	if errs := GetValidator().Validate(RangeReq{Age: 30, Workers: 3}); errs != nil {
		t.Fatalf("in-range should pass, got %v", errs)
	}
}
