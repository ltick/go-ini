package ini_test

import (
	"errors"
	. "gopkg.in/check.v1"
	"math"
	"reflect"

	"go-ini"
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
		map[string]bool{"v": true},
	}, {
		"v = 10",
		map[string]interface{}{"v": 10},
	}, {
		"v= 0b10",
		map[string]uint{"v": 2},
	}, {
		"v= 0xA",
		map[string]int{"v": 10},
	}, {
		"v= 4294967296",
		map[string]int64{"v": 4294967296},
	}, {
		"v= 0.1",
		map[string]interface{}{"v": 0.1},
	}, {
		"v= .1",
		map[string]float64{"v": 0.1},
	}, {
		"v= -10",
		map[string]int{"v": -10},
	}, {
		"v= -.1",
		map[string]float32{"v": -0.1},
	},

	// Floats from spec
	{
		"canonical= 6.8523e+5",
		map[string]interface{}{"canonical": 6.8523e+5},
	}, {
		"expo= 685.230_15e+03",
		map[string]float64{"expo": 685.23015e+03},
	}, {
		"fixed= 685_230.15",
		map[string]float32{"fixed": 685230.15},
	}, {
		"fixed= 685_230.15",
		map[string]interface{}{"fixed": 685230.15},
	},

	// Bools from spec
	{
		"canonical= y",
		map[string]bool{"canonical": true},
	}, {
		"answer= NO",
		map[string]interface{}{"answer": false},
	}, {
		"logical= True",
		map[string]bool{"logical": true},
	}, {
		"option= on",
		map[string]interface{}{"option": true},
	}, {
		"option= on",
		map[string]bool{"option": true},
	},
	// Ints from spec
	{
		"canonical= 685230",
		map[string]int{"canonical": 685230},
	}, {
		"decimal= +685_230",
		map[string]interface{}{"decimal": 685230},
	}, {
		"octal = 02472256",
		map[string]uint{"octal": 685230},
	}, {
		"hexa =  0x_0A_74_AE",
		map[string]int{"hexa": 685230},
	}, {
		"bin= 0b1010_0111_0100_1010_1110",
		map[string]uint{"bin": 685230},
	}, {
		"bin= -0b101010",
		map[string]int{"bin": -42},
	}, {
		"decimal= +685_230",
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
		map[string]interface{}{"1": "\"2\""},
	}, {
		"v= 'B'",
		map[string]interface{}{"v": "B"},
	}, {
		"hello.1= world_1",
		map[string]map[int]interface{}{
			"hello": map[int]interface{}{
				1: "world_1",
			},
		},
	}, {
		"hello.1.2= world_1_2",
		map[string]map[int]map[int]string{
			"hello": map[int]map[int]string{
				1: map[int]string{
					2: "world_1_2",
				},
			},
		},
	}, {
		"hello= world\nhello.1= world_1",
		map[string]map[int]string{
			"hello": map[int]string{
				1: "world_1",
			},
		},
	}, {
		"hello.1= world\nhello.1.2= world_1_2",
		map[string]map[int]map[int]string{
			"hello": map[int]map[int]string{
				1: map[int]string{
					2: "world_1_2",
				},
			},
		},
	},
	// section conversions.
	{
		"[section]\n'hello'= \"world\"",
		map[string]map[string]string{"section": map[string]string{"hello": "world"}},
	}, {
		"#comment\n[section]\nhello= world",
		map[string]map[string]string{"section": map[string]string{"hello": "world"}},
	}, {
		"hello_= world[section]\nhello= world",
		map[string]interface{}{
			"hello_": "world[section]",
			"hello":  "world",
		},
	}, {
		"hello_= world\n[section]\nhello= world",
		map[string]interface{}{
			"hello_": "world",
			"section": map[interface{}]interface{}{
				"hello_": "world",
				"hello":  "world",
			},
		},
	}, {
		"hello= world\n[section_1]\nhello_1= world\n[section_2:section_1]\nhello_2= world",
		map[string]interface{}{
			"hello": "world",
			"section_1": map[interface{}]interface{}{
				"hello":   "world",
				"hello_1": "world",
			},
			"section_2": map[interface{}]interface{}{
				"hello":   "world",
				"hello_1": "world",
				"hello_2": "world",
			},
		},
	}, {
		"hello.1= world\n[section]\nhello.2= world",
		map[string]interface{}{
			"hello": map[interface{}]interface{}{
				1: "world",
			},
			"section": map[interface{}]interface{}{
				"hello": map[interface{}]interface{}{
					1: "world",
					2: "world",
				},
			},
		},
	}, {
		"hello.1= world\n[section]\nhello.1.2= world",
		map[string]map[interface{}]interface{}{
			"hello": map[interface{}]interface{}{
				1: "world",
			},
			"section": map[interface{}]interface{}{
				"hello": map[interface{}]interface{}{
					1: map[interface{}]interface{}{
						2: "world",
					},
				},
			},
		},
	}, {
		"hello.1.2= world\n[section]\nhello.1= world",
		map[string]map[interface{}]interface{}{
			"hello": map[interface{}]interface{}{
				1: map[interface{}]interface{}{
					2: "world",
				},
			},
			"section": map[interface{}]interface{}{
				"hello": map[interface{}]interface{}{
					1: "world",
				},
			},
		},
	}, {
		"hello.1.2= world\nHello=1\n[section]\nhello.1= world",
		map[string]interface{}{
			"Hello": 1,
			"hello": map[interface{}]interface{}{
				1: map[interface{}]interface{}{
					2: "world",
				},
			}, "section": map[interface{}]interface{}{
				"Hello": 1,
				"hello": map[interface{}]interface{}{
					1: "world",
				},
			},
		},
	}, {
		"hello_1= world\n[section]\nhello_2.1= world\nhello_2.2.1= world_1\nhello_2.2.2= world_2",
		map[string]interface{}{
			"hello_1": "world",
			"section": map[interface{}]interface{}{
				"hello_1": "world",
				"hello_2": map[interface{}]interface{}{
					1: "world",
					2: map[interface{}]interface{}{
						1: "world_1",
						2: "world_2",
					},
				},
			},
		},
	},
	// struct conversions
	{
		"hello= world",
		&struct {
			Hello string
		}{Hello: "world"},
	}, {
		"[section]\nhello= world",
		&struct {
			Section *struct {
				Hello string
			}
		}{&struct{ Hello string }{"world"}},
	}, {
		"hello= world\n[section]\nhello_= world_1",
		&struct {
			Hello   string
			Section *struct {
				Hello_ string
			}
		}{"world", &struct{ Hello_ string }{"world_1"}},
	}, {
		"hello= world\n[section]\nhello_= world_1",
		&struct {
			Hello   string
			Section *struct {
				Hello_ string
			}
		}{"world", &struct{ Hello_ string }{"world_1"}},
	}, {
		"hello.a= world\n[section]\nhello.a.b= world",
		&struct {
			Hello struct {
				A string
			}
			Section struct {
				Hello struct{ A struct{ B string } }
			}
		}{
			struct{ A string }{"world"},
			struct {
				Hello struct{ A struct{ B string } }
			}{struct{ A struct{ B string } }{struct{ B string }{"world"}}},
		},
	}, {
		"hello.a= world\n[section]\nhello.a.b= world",
		&struct {
			Hello struct {
				A string
			}
			Section struct {
				Hello struct {
					A struct {
						B string
					}
				}
			}
		}{
			struct{ A string }{"world"},
			struct {
				Hello struct{ A struct{ B string } }
			}{struct{ A struct{ B string } }{struct{ B string }{"world"}}},
		},
	},
}

type M map[interface{}]interface{}

func (s *S) TestUnmarshal(c *C) {
	for _, item := range unmarshalTests {
		typ := reflect.ValueOf(item.value).Type()
		var value interface{}
		switch typ.Kind() {
		case reflect.Map:
			value = reflect.MakeMap(typ).Interface()
		case reflect.String:
			value = reflect.New(typ).Interface()
		case reflect.Ptr:
			value = reflect.New(typ.Elem()).Interface()
		default:
			c.Fatalf("missing case for %s", typ)
		}
		err := ini.Unmarshal([]byte(item.data), value)
		//fmt.Println("---")
		//fmt.Println(item.data)
		//fmt.Println(value)
		//fmt.Println("===")
		if _, ok := err.(*ini.TypeError); !ok {
			c.Assert(err, IsNil)
		}
		if typ.Kind() == reflect.String {
			c.Assert(*value.(*string), Equals, item.value)
		} else {
			c.Assert(value, DeepEquals, item.value)
		}
	}
}

func (s *S) TestUnmarshalNaN(c *C) {
	var value map[string]interface{}
	err := ini.Unmarshal([]byte("notanum= .NaN"), &value)
	//fmt.Println("---")
	//fmt.Println(value)
	//fmt.Println("===")
	c.Assert(err, IsNil)
	c.Assert(math.IsNaN(value["notanum"].(float64)), Equals, true)
}

var unmarshalErrorTests = []struct {
	data, error string
}{
	{
		"hello: world",
		"ini: line 1: did not find expected <value> or <map>",
	},
	{
		"[section]'hello'= \"world\"",
		"ini: must have a line break before the first section key",
	},
	{
		"hello= world\n[section_2:section_1]\nhello_2= world\n[section_1]\nhello_1= world",
		"ini: inherit section 'section_1' does not exists",
	},
}

func (s *S) TestUnmarshalErrors(c *C) {
	for _, item := range unmarshalErrorTests {
		var value interface{}
		err := ini.Unmarshal([]byte(item.data), &value)
		c.Assert(err, ErrorMatches, item.error, Commentf("Partial unmarshal: %#v", value))
	}
}

var unmarshalerTests = []struct {
	data  string
	value interface{}
}{
	{"hi=there", map[interface{}]interface{}{"hi": "there"}},
}

var unmarshalerResult = map[int]error{}

type unmarshalerType struct {
	value interface{}
}

func (o *unmarshalerType) UnmarshalINI(unmarshal func(v interface{}) error) error {
	if err := unmarshal(&o.value); err != nil {
		return err
	}
	if i, ok := o.value.(int); ok {
		if result, ok := unmarshalerResult[i]; ok {
			return result
		}
	}
	return nil
}

func (s *S) TestUnmarshalerWholeDocument(c *C) {
	obj := &unmarshalerType{}
	err := ini.Unmarshal([]byte(unmarshalerTests[0].data), obj)
	c.Assert(err, IsNil)
	value, ok := obj.value.(map[interface{}]interface{})
	//fmt.Println("---")
	//fmt.Println(value)
	//fmt.Println("===")
	c.Assert(ok, Equals, true, Commentf("value: %#v", obj.value))
	c.Assert(value, DeepEquals, unmarshalerTests[0].value)
}
