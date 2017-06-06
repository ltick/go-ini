package ini_test

import (
	"errors"
	"reflect"
	"testing"
	"tick-config-ini"
	"fmt"
	"strconv"
)

var failingErr = errors.New("failingErr")

var unmarshalIntTest = 123

var unmarshalTests = []struct {
	data  string
	value interface{}
}{
	{
		"",
		&struct{}{},
	}, {
		"v= hi",
		map[string]string{"v": "hi"},
	}, {
        "v= hi",
        map[string]interface{}{"v": "hi"},
    }, {
        "v = true",
        map[string]string{"v": "true"},
    }, {
        "v =true",
        map[string]interface{}{"v": true},
    }, {
        "v = 10",
        map[string]interface{}{"v": 10},
    }, {
        "v= 0b10",
        map[string]interface{}{"v": 2},
    }, {
        "v= 0xA",
        map[string]interface{}{"v": 10},
    }, {
        "v= 4294967296",
        map[string]int64{"v": 4294967296},
    }, {
        "v= 0.1",
        map[string]interface{}{"v": 0.1},
    }, {
        "v= .1",
        map[string]interface{}{"v": 0.1},
    }, {
        "v= -10",
        map[string]interface{}{"v": -10},
    }, {
        "v= -.1",
        map[string]interface{}{"v": -0.1},
        /*
    },

    // Simple values.
    {
        "123",
        &unmarshalIntTest,
    },

    // Floats from spec
    {
        "canonical= 6.8523e+5",
        map[string]interface{}{"canonical": 6.8523e+5},
    }, {
        "expo= 685.230_15e+03",
        map[string]interface{}{"expo": 685.23015e+03},
    }, {
        "fixed= 685_230.15",
        map[string]interface{}{"fixed": 685230.15},
    }, {
        "fixed= 685_230.15",
        map[string]float64{"fixed": 685230.15},
    },

    // Bools from spec
    {
        "canonical= y",
        map[string]interface{}{"canonical": true},
    }, {
        "answer:= NO",
        map[string]interface{}{"answer": false},
    }, {
        "logical= True",
        map[string]interface{}{"logical": true},
    }, {
        "option= on",
        map[string]interface{}{"option": true},
    }, {
        "option= on",
        map[string]bool{"option": true},
    },
    // Ints from spec
    {
        "canonical: 685230",
        map[string]interface{}{"canonical": 685230},
    }, {
        "decimal: +685_230",
        map[string]interface{}{"decimal": 685230},
    }, {
        "octal: 02472256",
        map[string]interface{}{"octal": 685230},
    }, {
        "hexa: 0x_0A_74_AE",
        map[string]interface{}{"hexa": 685230},
    }, {
        "bin: 0b1010_0111_0100_1010_1110",
        map[string]interface{}{"bin": 685230},
    }, {
        "bin: -0b101010",
        map[string]interface{}{"bin": -42},
    }, {
        "decimal: +685_230",
        map[string]int{"decimal": 685230},
    },

    // Nulls from spec
    {
        "empty=",
        map[string]interface{}{"empty": nil},
    }, {
        "canonical= ~",
        map[string]interface{}{"canonical": nil},
    }, {
        "english= null",
        map[string]interface{}{"english": nil},
    }, {
        "~= null key",
        map[interface{}]string{nil: "null key"},
    }, {
        "empty=",
        map[string]*bool{"empty": nil},
    },

    // section conversions.
    {
        "[section]\nhello= world",
        map[string]interface{}{"section": map[string]string{"hello": "world"}},
    }, {
        "hello1= world[section:default]\nhello= world",
        map[string]interface{}{"default": map[string]string{"hello1": "world"}, "section": map[string]string{"hello1": "world", "hello": "world"}},
    },

    // Some cross type conversions
    {
        "v= 42",
        map[string]uint{"v": 42},
    }, {
        "v= -42",
        map[string]uint{},
    }, {
        "v= 4294967296",
        map[string]uint64{"v": 4294967296},
    }, {
        "v= -4294967296",
        map[string]uint64{},
    },

    // int
    {
        "int_max= 2147483647",
        map[string]int{"int_max": math.MaxInt32},
    },
    {
        "int_min= -2147483648",
        map[string]int{"int_min": math.MinInt32},
    },
    {
        "int_overflow= 9223372036854775808", // math.MaxInt64 + 1
        map[string]int{},
    },

    // int64
    {
        "int64_max= 9223372036854775807",
        map[string]int64{"int64_max": math.MaxInt64},
    },
    {
        "int64_max_base2= 0b111111111111111111111111111111111111111111111111111111111111111",
        map[string]int64{"int64_max_base2": math.MaxInt64},
    },
    {
        "int64_min= -9223372036854775808",
        map[string]int64{"int64_min": math.MinInt64},
    },
    {
        "int64_neg_base2= -0b111111111111111111111111111111111111111111111111111111111111111",
        map[string]int64{"int64_neg_base2": -math.MaxInt64},
    },
    {
        "int64_overflow= 9223372036854775808", // math.MaxInt64 + 1
        map[string]int64{},
    },

    // uint
    {
        "uint_min= 0",
        map[string]uint{"uint_min": 0},
    },
    {
        "uint_max= 4294967295",
        map[string]uint{"uint_max": math.MaxUint32},
    },
    {
        "uint_underflow= -1",
        map[string]uint{},
    },

    // uint64
    {
        "uint64_min= 0",
        map[string]uint{"uint64_min": 0},
    },
    {
        "uint64_max= 18446744073709551615",
        map[string]uint64{"uint64_max": math.MaxUint64},
    },
    {
        "uint64_max_base2= 0b1111111111111111111111111111111111111111111111111111111111111111",
        map[string]uint64{"uint64_max_base2": math.MaxUint64},
    },
    {
        "uint64_maxint64= 9223372036854775807",
        map[string]uint64{"uint64_maxint64": math.MaxInt64},
    },
    {
        "uint64_underflow= -1",
        map[string]uint64{},
    },

    // float32
    {
        "float32_max= 3.40282346638528859811704183484516925440e+38",
        map[string]float32{"float32_max": math.MaxFloat32},
    },
    {
        "float32_nonzero= 1.401298464324817070923729583289916131280e-45",
        map[string]float32{"float32_nonzero": math.SmallestNonzeroFloat32},
    },
    {
        "float32_maxuint64= 18446744073709551615",
        map[string]float32{"float32_maxuint64": float32(math.MaxUint64)},
    },
    {
        "float32_maxuint64+1= 18446744073709551616",
        map[string]float32{"float32_maxuint64+1": float32(math.MaxUint64 + 1)},
    },

    // float64
    {
        "float64_max= 1.797693134862315708145274237317043567981e+308",
        map[string]float64{"float64_max": math.MaxFloat64},
    },
    {
        "float64_nonzero= 4.940656458412465441765687928682213723651e-324",
        map[string]float64{"float64_nonzero": math.SmallestNonzeroFloat64},
    },
    {
        "float64_maxuint64= 18446744073709551615",
        map[string]float64{"float64_maxuint64": float64(math.MaxUint64)},
    },
    {
        "float64_maxuint64+1= 18446744073709551616",
        map[string]float64{"float64_maxuint64+1": float64(math.MaxUint64 + 1)},
    },

    // Overflow cases.
    {
        "v= 4294967297",
        map[string]int32{},
    }, {
        "v= 128",
        map[string]int8{},
    },

    // Quoted values.
    {
        "'1'= '\"2\"'",
        map[interface{}]interface{}{"1": "\"2\""},
    }, {
        "v= 'B'",
        map[string]interface{}{"v": "B"},
*/
	},
}

type M map[interface{}]interface{}

func TestUnmarshal(t *testing.T) {
	for index, item := range unmarshalTests {
		fmt.Println("Test" + strconv.Itoa(index))
		itemValue := reflect.ValueOf(item.value)
		var value interface{}
		var typ reflect.Type = itemValue.Type()
		switch typ.Kind() {
		case reflect.Map:
			value = reflect.MakeMap(typ).Interface()
		case reflect.String:
			value = reflect.New(typ).Interface()
		case reflect.Ptr:
			value = reflect.New(typ.Elem()).Interface()
		default:
			t.Fatalf("missing case for %s", typ)
		}
		err := ini.Unmarshal([]byte(item.data), value)
		fmt.Println(err)
		if err != nil {
			t.Error("TestUnmarshal Failed! " + err.Error())
		}
		fmt.Println(value)
		fmt.Println(item.value)
		if !reflect.DeepEqual(value, item.value) {
			t.Errorf("%v and %v does not equal!", value, item.value)
		}
		/*
			if typ.Kind() == reflect.String {
				if *value.(*string) != item.value {
					t.Error("TestUnmarshal Failed")
				}
			} else if typ.Kind() == reflect.Map {
				unmarshalValue := value.(map[string]string)
				for mapKey, mapValue := range item.value.(map[string]string) {
					unmarshalValueItem, ok := unmarshalValue[mapKey]
					if !ok || unmarshalValueItem != mapValue {
						t.Error("TestUnmarshal Failed")
					}
				}
			} else {
				if value != item.value {
					t.Error("TestUnmarshal Failed")
				}
			}
		*/
	}
}

/*
func TestUnmarshalNaN(t *testing.T) {
	value := map[string]interface{}{}
	err := ini.Unmarshal([]byte("notanum: .NaN"), &value)
	assert.Nil(t, err)
	assert.Equal(t, math.IsNaN(value["notanum"].(float64)), true)
}
*/
