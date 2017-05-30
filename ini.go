package ini

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
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
	defaultSection     = "common"    // default section means if some ini items not in a section, make them in default section,
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

func GetDefaultSection() string {
	return defaultSection
}

func Unmarshal(in []byte, out interface{}) (err error) {
	defer handleErr(&err)
	d := newDecoder()
	p := newParser(in)
	defer p.destroy()
	node := p.parse()
	fmt.Println("node:")
	fmt.Println(node)
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

	// InlineMap is the number of the field in the struct that
	// contains an ,inline map, or -1 if there's none.
	InlineMap int
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
	inlineMap := -1
	for i := 0; i != n; i++ {
		field := st.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue // Private field
		}

		info := fieldInfo{Num: i}

		tag := field.Tag.Get("yaml")
		if tag == "" && strings.Index(string(field.Tag), ":") < 0 {
			tag = string(field.Tag)
		}
		if tag == "-" {
			continue
		}

		inline := false
		fields := strings.Split(tag, ",")
		if len(fields) > 1 {
			for _, flag := range fields[1:] {
				switch flag {
				case "omitempty":
					info.OmitEmpty = true
				case "flow":
					info.Flow = true
				case "inline":
					inline = true
				default:
					return nil, errors.New(fmt.Sprintf("Unsupported flag %q in tag %q of type %s", flag, tag, st))
				}
			}
			tag = fields[0]
		}

		if inline {
			switch field.Type.Kind() {
			case reflect.Map:
				if inlineMap >= 0 {
					return nil, errors.New("Multiple ,inline maps in struct " + st.String())
				}
				if field.Type.Key() != reflect.TypeOf("") {
					return nil, errors.New("Option ,inline needs a map with string keys in struct " + st.String())
				}
				inlineMap = info.Num
			case reflect.Struct:
				sinfo, err := getStructInfo(field.Type)
				if err != nil {
					return nil, err
				}
				for _, finfo := range sinfo.FieldsList {
					if _, found := fieldsMap[finfo.Key]; found {
						msg := "Duplicated key '" + finfo.Key + "' in struct " + st.String()
						return nil, errors.New(msg)
					}
					if finfo.Inline == nil {
						finfo.Inline = []int{i, finfo.Num}
					} else {
						finfo.Inline = append([]int{i}, finfo.Inline...)
					}
					fieldsMap[finfo.Key] = finfo
					fieldsList = append(fieldsList, finfo)
				}
			default:
				//return nil, errors.New("Option ,inline needs a struct value or map field")
				return nil, errors.New("Option ,inline needs a struct value field")
			}
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

	sinfo = &structInfo{fieldsMap, fieldsList, inlineMap}

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

/*old*/

func unmarshal(content []byte, data reflect.Value, comment reflect.Value) (err error) {
	var copyData func(src reflect.Value, dst reflect.Value) (err error)
	copyData = func(src reflect.Value, dst reflect.Value) (err error) {
		if src.Kind() == reflect.Map {
			for _, key := range src.MapKeys() {
				value := src.MapIndex(key)
				switch value.Kind() {
				case reflect.Map:
					subValue := reflect.MakeMap(itemType)
					err = copyData(value, subValue)
					if err != nil {
						dst.SetMapIndex(key, subValue)
					}
				default:
					dst.SetMapIndex(key, value)
				}
			}
		}
		return nil
	}

	var addData func(key string, value string, data reflect.Value) (err error)
	addData = func(key string, value string, data reflect.Value) (err error) {
		v := data
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			v = v.Elem()
		}
		keys := strings.Split(key, ".")
		key = keys[0]
		if len(keys) == 1 {
			v.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		} else {
			subValue := reflect.MakeMap(itemType)
			tmpValue := v.MapIndex(reflect.ValueOf(key))
			if tmpValue.IsValid() {
				if reflect.TypeOf(tmpValue) == itemType {
					subValue = tmpValue
				}
			}
			err = addData(strings.Join(keys[1:], "."), value, reflect.ValueOf(&subValue))
			if err != nil {
				return err
			}
		}
		return nil
	}

	var commentBuf bytes.Buffer
	buf := bufio.NewReader(bytes.NewReader(content))
	section := defaultSection
	implementSection := ""
	for {
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		if bytes.Equal(line, bEmpty) {
			continue
		}
		line = bytes.TrimSpace(line)

		var bComment []byte
		switch {
		case bytes.HasPrefix(line, bNumComment):
			bComment = bNumComment
		case bytes.HasPrefix(line, bSemComment):
			bComment = bSemComment
		}
		if bComment != nil {
			line = bytes.TrimLeft(line, string(bComment))
			// Need append to a new line if multi-line comments.
			if commentBuf.Len() > 0 {
				commentBuf.WriteByte('\n')
			}
			commentBuf.Write(line)
			continue
		}
		if bytes.HasPrefix(line, sectionStart) && bytes.HasSuffix(line, sectionEnd) {
			section = strings.ToLower(string(line[1 : len(line)-1])) // section name case insensitive
			if seperatorIndex := bytes.Index(line, []byte(implementSeperator)); seperatorIndex != -1 {
				implementSection = string(line[seperatorIndex+1 : len(line)-1])
				section = string(line[1:seperatorIndex])
			}
			if commentBuf.Len() > 0 {
				comment.Elem().MapIndex(reflect.ValueOf(section)).SetString(commentBuf.String())
				commentBuf.Reset()
			}
			continue
		}
		sectionData := data.MapIndex(reflect.ValueOf(section))
		// data[section] not exists
		if !sectionData.IsValid() {
			data.SetMapIndex(reflect.ValueOf(section), reflect.MakeMap(itemType))
		}
		implementSectionData := data.MapIndex(reflect.ValueOf(implementSection))
		// data[implementSection] exists
		if implementSectionData.IsValid() {
			sectionData = reflect.MakeMap(itemType)
			copyData(sectionData, implementSectionData)
			data.SetMapIndex(reflect.ValueOf(section), sectionData)
		}

		keyValue := bytes.SplitN(line, bEqual, 2)

		key := string(bytes.TrimSpace(keyValue[0])) // key name case insensitive
		key = strings.ToLower(key)

		if len(keyValue) != 2 {
			return errors.New("read the content error: \"" + string(line) + "\", should key = val")
		}
		val := bytes.TrimSpace(keyValue[1])
		if bytes.HasPrefix(val, bDQuote) {
			val = bytes.Trim(val, `"`)
		}
		err = addData(key, string(val), sectionData.Addr())
		if err != nil {
			return err
		}
		data.SetMapIndex(reflect.ValueOf(section), sectionData)

		if commentBuf.Len() > 0 {
			comment.SetMapIndex(reflect.ValueOf(section+"."+key), reflect.ValueOf(commentBuf.String()))
			commentBuf.Reset()
		}

		implementSection = ""
	}

	return nil

}

// Set writes a new value for key.
// if write to one section, the key need be "section::key".
/*
func (c *IniConfig) Set(key string, value string) (err error) {
	c.Lock()
	defer c.Unlock()

	if key == "" {
		return errors.New("key is empty")
	}

	keys := strings.Split(strings.ToLower(key), "::")
	section := ""
	if len(keys) >= 2 {
		section = keys[0]
		key = keys[1]
	} else {
		section = defaultSection
		key = keys[0]
	}

	if _, ok := c.data[section]; !ok {
		c.data[section] = make(map[string]interface{})
	}
	keys = strings.Split(key, ".")
	key_len := len(keys)

	data := c.data[section]
	for index, key := range keys[0:] {
		if index == key_len-1 {
			data[key] = value
		} else {
			if _, ok := data[key]; ok {
				data, ok = data[key].(map[string]interface{})
				if !ok {
					return errors.New("not exist key:" + key)
				}
			} else {
				return errors.New("not exist key:" + key)
			}
		}
	}

	return nil
}
*/
