package output

import (
	"reflect"
	"regexp"
)

// ansiEscape matches ANSI CSI escape sequences (colors, cursor moves, etc.).
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func stripANSIText(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func sanitizeANSIData(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	return sanitizeANSIReflect(reflect.ValueOf(v)).Interface()
}

func sanitizeANSIReflect(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}

	switch v.Kind() {
	case reflect.String:
		s := stripANSIText(v.String())
		return reflect.ValueOf(s).Convert(v.Type())
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(sanitizeANSIReflect(v.Elem()))
		return out
	case reflect.Interface:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		sanitized := sanitizeANSIReflect(v.Elem())
		out := reflect.New(v.Type()).Elem()
		if sanitized.IsValid() && sanitized.Type().AssignableTo(v.Type()) {
			out.Set(sanitized)
			return out
		}
		if sanitized.IsValid() {
			out.Set(reflect.ValueOf(sanitized.Interface()))
		}
		return out
	case reflect.Slice:
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(sanitizeANSIReflect(v.Index(i)))
		}
		return out
	case reflect.Array:
		out := reflect.New(v.Type()).Elem()
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(sanitizeANSIReflect(v.Index(i)))
		}
		return out
	case reflect.Map:
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			val := sanitizeANSIReflect(iter.Value())
			if val.IsValid() && val.Type().AssignableTo(v.Type().Elem()) {
				out.SetMapIndex(k, val)
				continue
			}
			if val.IsValid() && val.Type().ConvertibleTo(v.Type().Elem()) {
				out.SetMapIndex(k, val.Convert(v.Type().Elem()))
				continue
			}
			out.SetMapIndex(k, iter.Value())
		}
		return out
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			dst := out.Field(i)
			if !dst.CanSet() {
				continue
			}
			src := v.Field(i)
			sanitized := sanitizeANSIReflect(src)
			if sanitized.IsValid() && sanitized.Type().AssignableTo(dst.Type()) {
				dst.Set(sanitized)
			} else if sanitized.IsValid() && sanitized.Type().ConvertibleTo(dst.Type()) {
				dst.Set(sanitized.Convert(dst.Type()))
			} else {
				dst.Set(src)
			}
		}
		return out
	default:
		return v
	}
}
