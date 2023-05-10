package utils

import "reflect"

func TypeString[T any]() string {
	var placeholder T
	return reflect.TypeOf(placeholder).Name()
}
