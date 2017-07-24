package ini

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// MapSlice encodes and decodes as a INI map.
// The order of keys is preserved when encoding and decoding.
type MapSlice []MapItem

// MapItem is an item in a MapSlice.
type MapItem struct {
	Key, Value interface{}
}

// The Unmarshaler interface may be implemented by types to customize their
// behavior when being unmarshaled from a INI document. The UnmarshalINI
// method receives a function that may be called to unmarshal the original
// INI value into a field or variable. It is safe to call the unmarshal
// function parameter more than once if necessary.
type Unmarshaler interface {
	UnmarshalINI(unmarshal func(interface{}) error) error
}

// The Marshaler interface may be implemented by types to customize their
// behavior when being marshaled into a INI document. The returned value
// is marshaled in place of the original value implementing Marshaler.
//
// If an error is returned by MarshalINI, the marshaling procedure stops
// and returns with the provided error.
type Marshaler interface {
	MarshalINI() (interface{}, error)
}

var (
	bNumComment        = []byte{'#'} // number signal
	bSemComment        = []byte{';'} // semicolon signal
	bEmpty             = []byte{}
	bEqual             = []byte{'='} // equal signal
	bDQuote            = []byte{'"'} // quote signal
	sectionStart       = []byte{'['} // section start signal
	sectionEnd         = []byte{']'} // section end signal
	implementSeperator = []byte{':'} // section implement signal
	lineBreak          = "\n"

	itemType = reflect.TypeOf(map[string]interface{}{})
)

func Unmarshal(in []byte, out interface{}) (err error) {
	defer handleErr(&err)
	d := newDecoder()
	p := newParser(in)
	defer p.destroy()
	node := p.parse()
	if node != nil {
		v := reflect.ValueOf(out)
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			v = v.Elem()
		}
		d.unmarshal(node, v)
	}
	if len(d.terrors) > 0 {
		return &TypeError{d.terrors}
	}
	return nil
}

func Marshal(in interface{}) (out []byte, err error) {
	defer handleErr(&err)
	e := newEncoder()
	defer e.destroy()
	e.marshal(reflect.ValueOf(in))
	e.finish()
	out = e.out
	return
}

func handleErr(err *error) {
	if v := recover(); v != nil {
		if e, ok := v.(iniError); ok {
			*err = e.err
		} else {
			panic(v)
		}
	}
}

type iniError struct {
	err error
}

func fail(err error) {
	panic(iniError{err})
}

func failf(format string, args ...interface{}) {
	panic(iniError{fmt.Errorf("ini: "+format, args...)})
}

// A TypeError is returned by Unmarshal when one or more fields in
// the INI document cannot be properly decoded into the requested
// types. When this error is returned, the value is still
// unmarshaled partially.
type TypeError struct {
	Errors []string
}

func (e *TypeError) Error() string {
	return fmt.Sprintf("ini: unmarshal errors:\n  %s", strings.Join(e.Errors, "\n  "))
}

// --------------------------------------------------------------------------
// Maintain a mapping of keys to structure field indexes

// The code in this section was copied from mgo/bson.

// structInfo holds details for the serialization of fields of
// a given struct.
type structInfo struct {
	FieldsMap  map[string]fieldInfo
	FieldsList []fieldInfo
}

type fieldInfo struct {
	Key       string
	Num       int
	OmitEmpty bool
	Flow      bool

	// Inline holds the field index if the field is part of an inlined struct.
	Inline []int
}

var structMap = make(map[reflect.Type]*structInfo)
var fieldMapMutex sync.RWMutex

func getStructInfo(st reflect.Type) (*structInfo, error) {
	fieldMapMutex.RLock()
	sinfo, found := structMap[st]
	fieldMapMutex.RUnlock()
	if found {
		return sinfo, nil
	}

	n := st.NumField()
	fieldsMap := make(map[string]fieldInfo)
	fieldsList := make([]fieldInfo, 0, n)
	for i := 0; i != n; i++ {
		field := st.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue // Private field
		}

		info := fieldInfo{Num: i}

		tag := field.Tag.Get("ini")
		if tag == "" && strings.Index(string(field.Tag), ":") < 0 {
			tag = string(field.Tag)
		}
		if tag == "-" {
			continue
		}

		if tag != "" {
			info.Key = tag
		} else {
			info.Key = strings.ToLower(field.Name)
		}

		if _, found = fieldsMap[info.Key]; found {
			msg := "Duplicated key '" + info.Key + "' in struct " + st.String()
			return nil, errors.New(msg)
		}

		fieldsList = append(fieldsList, info)
		fieldsMap[info.Key] = info
	}

	sinfo = &structInfo{fieldsMap, fieldsList}

	fieldMapMutex.Lock()
	structMap[st] = sinfo
	fieldMapMutex.Unlock()
	return sinfo, nil
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return len(v.String()) == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Slice:
		return v.Len() == 0
	case reflect.Map:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Struct:
		vt := v.Type()
		for i := v.NumField() - 1; i >= 0; i-- {
			if vt.Field(i).PkgPath != "" {
				continue // Private field
			}
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	}
	return false
}
