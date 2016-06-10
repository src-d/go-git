package cshared

import (
	"C"
	"reflect"
)

//export c_std_map_get_str_obj
func c_std_map_get_str_obj(m uint64, key string) uint64 {
	obj, ok := GetObject(Handle(m))
	if !ok {
		return IH
	}
	mapval := reflect.ValueOf(obj)
	if mapval.Type().Kind() != reflect.Map {
		return IH
	}
	val := mapval.MapIndex(reflect.ValueOf(key))
	if !val.IsValid() {
		return IH
	}
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.IsNil() {
		return IH
	}
	val_handle := RegisterObject(val.Interface())
	return uint64(val_handle)
}

//export c_std_map_get_obj_obj
func c_std_map_get_obj_obj(m uint64, key uint64) uint64 {
	obj, ok := GetObject(Handle(m))
	if !ok {
		return IH
	}
	mapval := reflect.ValueOf(obj)
	if mapval.Type().Kind() != reflect.Map {
		return IH
	}
	obj, ok = GetObject(Handle(key))
	if !ok {
		return IH
	}
	val := mapval.MapIndex(reflect.ValueOf(obj))
	if !val.IsValid() {
		return IH
	}
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.IsNil() {
		return IH
	}
	val_handle := RegisterObject(val.Interface())
	return uint64(val_handle)
}
