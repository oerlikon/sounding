package mainutil

import (
	"fmt"
	"reflect"
	"strconv"

	"gopkg.in/validator.v2"
)

func Validate(v interface{}) error {
	vt := validator.NewValidator()
	vt.SetTag("traits")
	vt.SetValidationFunc("nz", nz)
	vt.SetValidationFunc("gt", gt)
	vt.SetValidationFunc("ge", ge)
	vt.SetValidationFunc("lt", lt)
	vt.SetValidationFunc("le", le)
	errs, _ := vt.Validate(v).(validator.ErrorMap)
	for k, err := range errs {
		if len(err[0].Error()) > 0 {
			return fmt.Errorf("%s %s?", k, err)
		} else {
			return fmt.Errorf("%s?", k)
		}
	}
	return nil
}

func nz(v interface{}, _ string) error {
	st, valid := reflect.ValueOf(v), true
	if st.Kind() == reflect.Ptr {
		if st.IsNil() {
			return nil
		}
		st = st.Elem()
	}
	switch st.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		valid = st.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		valid = st.Uint() != 0
	case reflect.Float32, reflect.Float64:
		valid = st.Float() != 0
	default:
		panic("mainutil.Validate: unsupported type")
	}
	if !valid {
		return fmt.Errorf("")
	}
	return nil
}

func gt(v interface{}, param string) error {
	st, valid := reflect.ValueOf(v), true
	if st.Kind() == reflect.Ptr {
		if st.IsNil() {
			return nil
		}
		st = st.Elem()
	}
	switch st.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := strconv.ParseInt(param, 0, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Int() > p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := strconv.ParseUint(param, 0, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Uint() > p
	case reflect.Float32, reflect.Float64:
		p, err := strconv.ParseFloat(param, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Float() > p
	default:
		panic("mainutil.Validate: unsupported type")
	}
	if !valid {
		return fmt.Errorf("<= %s", param)
	}
	return nil
}

func ge(v interface{}, param string) error {
	st, valid := reflect.ValueOf(v), true
	if st.Kind() == reflect.Ptr {
		if st.IsNil() {
			return nil
		}
		st = st.Elem()
	}
	switch st.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := strconv.ParseInt(param, 0, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Int() >= p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := strconv.ParseUint(param, 0, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Uint() >= p
	case reflect.Float32, reflect.Float64:
		p, err := strconv.ParseFloat(param, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Float() >= p
	default:
		panic("mainutil.Validate: unsupported type")
	}
	if !valid {
		return fmt.Errorf("< %s", param)
	}
	return nil
}

func lt(v interface{}, param string) error {
	st, valid := reflect.ValueOf(v), true
	if st.Kind() == reflect.Ptr {
		if st.IsNil() {
			return nil
		}
		st = st.Elem()
	}
	switch st.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := strconv.ParseInt(param, 0, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Int() < p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := strconv.ParseUint(param, 0, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Uint() < p
	case reflect.Float32, reflect.Float64:
		p, err := strconv.ParseFloat(param, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Float() < p
	default:
		panic("mainutil.Validate: unsupported type")
	}
	if !valid {
		return fmt.Errorf(">= %s", param)
	}
	return nil
}

func le(v interface{}, param string) error {
	st, valid := reflect.ValueOf(v), true
	if st.Kind() == reflect.Ptr {
		if st.IsNil() {
			return nil
		}
		st = st.Elem()
	}
	switch st.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := strconv.ParseInt(param, 0, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Int() <= p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := strconv.ParseUint(param, 0, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Uint() <= p
	case reflect.Float32, reflect.Float64:
		p, err := strconv.ParseFloat(param, 64)
		if err != nil {
			panic(fmt.Sprintf("mainutil.Validate: %s", err))
		}
		valid = st.Float() <= p
	default:
		panic("mainutil.Validate: unsupported type")
	}
	if !valid {
		return fmt.Errorf("> %s", param)
	}
	return nil
}
