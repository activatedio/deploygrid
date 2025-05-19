package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"reflect"
	"strings"
	"sync"
)

type Validating interface {
	DoValidate() error
}

type Error struct {
	Key      string `json:"key,omitempty"`
	TypeName string `json:"type_name,omitempty"`
	Detail   error  `json:"detail"`
}

func (e Error) Error() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("type: %s ", e.TypeName))

	if e.Key != "" {
		sb.WriteString(fmt.Sprintf("key: %s ", e.Key))
	}
	if e.Detail != nil {
		sb.WriteString(fmt.Sprintf("detail: %s", e.Detail.Error()))
	}

	return sb.String()
}

// MustUnmarshallAndValidate unmarhalls and validates
func MustUnmarshallAndValidate[T Validating](v *viper.Viper, key string, in T) T {

	in = MustUnmarshall(v, key, in)

	if verr := in.DoValidate(); verr != nil {
		panic(wrapError(in, key, verr))
	}

	return in
}

func wrapError[T any](in T, key string, err error) error {

	return Error{
		Key:      key,
		TypeName: reflect.TypeOf(in).Elem().Name(),
		Detail:   err,
	}
}

var (
	prefixCache = map[string]bool{}
	prefixLock  = sync.Mutex{}
)

func bindEnvs[T any](v *viper.Viper, key string, in T) {

	prefixLock.Lock()
	defer prefixLock.Unlock()

	if _, ok := prefixCache[key]; ok {
		return
	}

	upperUnderscore := func(s string) string {
		return strings.ReplaceAll(strings.ToUpper(s), ".", "_")
	}

	upKey := upperUnderscore(key)

	for _, _key := range getAllKeys(in) {

		_upKey := upperUnderscore(_key)
		var envKey string
		if key != "" {
			envKey = fmt.Sprintf("%s_%s", upKey, _upKey)
		} else {
			envKey = _upKey
		}
		val, ok := os.LookupEnv(envKey)
		if ok {
			if key != "" {
				v.Set(fmt.Sprintf("%s.%s", key, _key), val)
			} else {
				v.Set(_key, val)
			}
		}
	}

	prefixCache[key] = true

}

// MustUnmarshall unmarshalls without any validation
/*
To work well with environment variables, setup viper as follows:

v := viper.New()
v.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
v.AutomaticEnv()

*/
func MustUnmarshall[T any](v *viper.Viper, key string, in T) T {

	bindEnvs(v, key, in)

	var err error
	if key == "" {
		err = v.Unmarshal(in)
	} else {
		err = v.UnmarshalKey(key, in)
	}

	check(err)

	return in
}

// Adapted from
// https://github.com/spf13/viper/issues/761
// To get all keys even has zero value, nil pointer
func getAllKeys(iface interface{}, parts ...string) []string {
	var keys []string

	ifv := reflect.ValueOf(iface)
	if ifv.Kind() == reflect.Ptr {
		ifv = ifv.Elem()
	}

	for i := 0; i < ifv.NumField(); i++ {
		v := ifv.Field(i)
		t := ifv.Type().Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}

		switch v.Kind() {
		case reflect.Struct:
			keys = append(keys, getAllKeys(v.Interface(), append(parts, tv)...)...)
		case reflect.Ptr:
			if v.IsNil() && v.CanSet() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			if v.Elem().Kind() == reflect.Struct {
				keys = append(keys, getAllKeys(v.Interface(), append(parts, tv)...)...)
			}
			keys = append(keys, strings.Join(append(parts, tv), "."))
		default:
			keys = append(keys, strings.Join(append(parts, tv), "."))
		}
	}

	return keys
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
