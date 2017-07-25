package ini_test

/*
var marshalTests = []struct {
value interface{}
data  string
}{
{
    nil,
    "null\n",
}, {
    map[string]string{"v": "hi"},
    "v: hi\n",

        }, {
            map[string]interface{}{"v": "hi"},
            "v: hi\n",
        }, {
            map[string]string{"v": "true"},
            "v: \"true\"\n",
        }, {
            map[string]string{"v": "false"},
            "v: \"false\"\n",
        }, {
            map[string]interface{}{"v": true},
            "v: true\n",
        }, {
            map[string]interface{}{"v": false},
            "v: false\n",
        }, {
            map[string]interface{}{"v": 10},
            "v: 10\n",
        }, {
            map[string]interface{}{"v": -10},
            "v: -10\n",
        }, {
            map[string]uint{"v": 42},
            "v: 42\n",
        }, {
            map[string]interface{}{"v": int64(4294967296)},
            "v: 4294967296\n",
        }, {
            map[string]int64{"v": int64(4294967296)},
            "v: 4294967296\n",
        }, {
            map[string]uint64{"v": 4294967296},
            "v: 4294967296\n",
        }, {
            map[string]interface{}{"v": "10"},
            "v: \"10\"\n",
        }, {
            map[string]interface{}{"v": 0.1},
            "v: 0.1\n",
        }, {
            map[string]interface{}{"v": float64(0.1)},
            "v: 0.1\n",
        }, {
            map[string]interface{}{"v": -0.1},
            "v: -0.1\n",
        }, {
            map[string]interface{}{"v": math.Inf(+1)},
            "v: .inf\n",
        }, {
            map[string]interface{}{"v": math.Inf(-1)},
            "v: -.inf\n",
        }, {
            map[string]interface{}{"v": math.NaN()},
            "v: .nan\n",
        }, {
            map[string]interface{}{"v": nil},
            "v: null\n",
        }, {
            map[string]interface{}{"v": ""},
            "v: \"\"\n",
        }, {
            map[string]interface{}{"v": map[string]string{"0": "A", "1": "B"}},
            "v.0:A\nv.1:B\n",
        }, {
            map[string]interface{}{"v": map[string]interface{}{"0": "A", "1": map[string]string{"1": "B"}}},
            "v.0:A\nv.1.1:B\n",
        }, {
            map[string]interface{}{"v": map[string]interface{}{"0": "A", "1": map[string]string{"1": "B", "2": "C"}}},
            "v.0:A\nv.1.1:B\nv.1.2=C\n",
        }, {
            map[string]interface{}{"a": "="},
            "a='='",
        }, {
            map[string]interface{}{"a": "[A]"},
            "a='[A]'",
        }, {
            map[string]interface{}{"a": "[A:B]"},
            "a='[A:B]'",
	},
}

func TestMarshal(t *testing.T) {
	defer os.Setenv("TZ", os.Getenv("TZ"))
	os.Setenv("TZ", "UTC")
	for _, item := range marshalTests {
		data, err := ini.Marshal(item.value)
		if err != nil {
			t.Error("TestUnmarshal Failed")
		}
		if string(data) != item.data {
			t.Error("TestUnmarshal Failed")
		}
	}
}

type marshalerType struct {
	value interface{}
}

func (o marshalerType) MarshalText() ([]byte, error) {
	panic("MarshalText called on type with MarshalINI")
}

func (o marshalerType) MarshalINI() (interface{}, error) {
	return o.value, nil
}

type marshalerValue struct {
	Field marshalerType "_"
}

func TestMarshalerWholeDocument(t *testing.T) {
	obj := &marshalerType{}
	obj.value = map[string]string{"hello": "world!"}
	data, err := ini.Marshal(obj)
	if err != nil {
		t.Error("TestUnmarshal Failed")
	}
	if string(data) != "hello= world!\n" {
		t.Error("TestUnmarshal Failed")
	}
}

type failingMarshaler struct{}

func (ft *failingMarshaler) MarshalINI() (interface{}, error) {
	return nil, failingErr
}

func TestMarshalerError(t *testing.T) {
	_, err := ini.Marshal(&failingMarshaler{})
	if err != failingErr {
		t.Error("TestUnmarshal Failed")
	}
}
*/
