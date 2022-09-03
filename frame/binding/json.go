package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

type jsonBinding struct {
	DisallowUnknownFields bool
	IsValidate            bool
}

func (j jsonBinding) Name() string {
	return "json"
}
func (j jsonBinding) Bind(r *http.Request, obj any) error {
	//post传参的内容放在body中
	body := r.Body
	if body == nil {
		return errors.New("invalid request")
	}
	decoder := json.NewDecoder(body)
	//传入的参数里有某个值，但是提供的结构体里没有，返回错误
	if j.DisallowUnknownFields == true {
		decoder.DisallowUnknownFields()
	}
	//结构体中有的属性，参数中没有，错误校验
	if j.IsValidate {
		err := validateParam(obj, decoder)
		if err != nil {
			return err
		}
	} else {
		err := decoder.Decode(obj)
		if err != nil {
			return err
		}
	}
	return validate(obj)
}

func validate(obj any) error {
	return Validator.ValidateStruct(obj)
}

//json结构体校验
func validateParam(obj any, decoder *json.Decoder) error {
	valueOf := reflect.ValueOf(obj)
	//判断是否为指针类型
	if valueOf.Kind() != reflect.Pointer {
		return errors.New("This argument must have a pointer type")
	}
	//元素的类型
	elem := valueOf.Elem().Interface()
	of := reflect.ValueOf(elem)
	switch of.Kind() {
	case reflect.Struct:
		//解析为map，根据map中的key进行比对
		//判断类型，结构体 才能解析为map
		return checkParam(of, obj, decoder)
	case reflect.Slice, reflect.Array:
		elem := of.Type().Elem()
		elemType := elem.Kind()
		if elemType == reflect.Struct {
			return checkParamSlice(elem, obj, decoder)
		}
	default:
		err := decoder.Decode(obj)
		if err != nil {
			return err
		}

	}
	return nil
}

//切片结构体校验
func checkParamSlice(elem reflect.Type, data any, decoder *json.Decoder) error {
	mapData := make([]map[string]interface{}, 0)
	_ = decoder.Decode(&mapData)
	if len(mapData) <= 0 {
		return nil
	}
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		name := field.Name
		mustType := field.Tag.Get("must")
		tag := field.Tag.Get("json")
		if tag != "" {
			name = tag
		}
		//value := mapData[0][tag]
		for _, v := range mapData {
			value := v[name]
			if value == nil && mustType == "mustType" {
				return errors.New(fmt.Sprintf("filed [%s] is required,because [%s] is must", tag, tag))
			}
		}

	}
	if data != nil {
		marshal, _ := json.Marshal(mapData)
		_ = json.Unmarshal(marshal, data)
	}
	return nil
}

//结构体校验
func checkParam(value reflect.Value, data any, decoder *json.Decoder) error {
	mapData := make(map[string]interface{})
	_ = decoder.Decode(&mapData)
	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		name := field.Name
		mustType := field.Tag.Get("must")
		tag := field.Tag.Get("json")
		if tag != "" {
			name = tag
		}
		value := mapData[name]
		if value == nil && mustType == "mustType" {
			return errors.New(fmt.Sprintf("filed [%s] is required,because [%s] is must", tag))
		}
	}
	if data != nil {
		marshal, _ := json.Marshal(mapData)
		_ = json.Unmarshal(marshal, data)
	}
	return nil
}
