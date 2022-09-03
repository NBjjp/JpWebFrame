package binding

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
	"sync"
)

//引用第三方验证  可验证值缺失  值范围

// 在验证的时候，每次都需要使用`validator.New()`，会极大的浪费性能，
//可以使用单例来做优化，同时验证器的实现可能有多种，提供接口，便于扩展
type StructValidator interface {
	//结构体验证，如果错误返回对应的错误信息
	ValidateStruct(any) error
	//返回对应使用的验证器
	Engine() any
}

var Validator StructValidator = &defaultValidator{}

type defaultValidator struct {
	one      sync.Once
	validate *validator.Validate
}

type SliceValidationError []error

//如果返回多个错误，将所有错误换行打印
func (err SliceValidationError) Error() string {
	n := len(err)
	switch n {
	case 0:
		return ""
	default:
		var b strings.Builder
		if err[0] != nil {
			fmt.Fprintf(&b, "[%d]: %s", 0, err[0].Error())
		}
		if n > 1 {
			for i := 1; i < n; i++ {
				if err[i] != nil {
					b.WriteString("\n")
					fmt.Fprintf(&b, "[%d]: %s", i, err[i].Error())
				}
			}
		}
		return b.String()
	}
}

func (d *defaultValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}
	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		return d.ValidateStruct(value.Elem().Interface())
	case reflect.Struct:
		return d.validateStruct(obj)
	case reflect.Slice, reflect.Array:
		count := value.Len()
		validateRet := make(SliceValidationError, 0)
		for i := 0; i < count; i++ {
			if err := d.validateStruct(value.Index(i).Interface()); err != nil {
				validateRet = append(validateRet, err)
			}
		}
		if len(validateRet) == 0 {
			return nil
		}
		return validateRet
	default:
		return nil
	}
}

func (d *defaultValidator) Engine() any {
	d.lazyInit()
	return d.validate
}
func (d *defaultValidator) validateStruct(obj any) error {
	d.lazyInit()
	return d.validate.Struct(obj)
}

func (d *defaultValidator) lazyInit() {
	d.one.Do(func() {
		d.validate = validator.New()
	})
}
