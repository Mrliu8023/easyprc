package rreflect

import (
	"fmt"
	"reflect"
)

// GetAllFn get all methods of a struct.It returns a map.The key of map is the name the name of func.
func GetAllFn(s interface{}) (int, map[string]reflect.Value) {
	sv := reflect.TypeOf(s)
	mMap := make(map[string]reflect.Value)
	for i := 0; i < sv.NumMethod(); i++ {
		m := sv.Method(i)
		mMap[m.Name] = m.Func
	}
	return sv.NumMethod(), mMap
}

func Call(value reflect.Value, params []interface{}) ([]interface{}, error) {
	ps := make([]reflect.Value, 0, len(params))
	for _, p := range params {
		ps = append(ps, reflect.ValueOf(p))
	}
	vs := value.Call(ps)
	if err := recover(); err != nil {
		return nil, fmt.Errorf("call error: %+v", err)
	}
	results := make([]interface{}, 0, len(vs))
	for _, r := range vs {
		results = append(results, r.Interface())
	}
	return results, nil
}
