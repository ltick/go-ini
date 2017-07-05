package ini_test

import (
	. "gopkg.in/check.v1"

	"tick-config-ini"
)

func (s *S) TestIni(c *C) {
	var (
		iniContext = `;comment one
#comment two
[common]
string = testing
string_1 = testing_1
string_1.1 = "testing"
string_2.1 = "testing"
string_3.1.1 = "testing_1"
string_3.1.2 = "testing_2"
iNt = 8080
Float = 3.1415976
BOOLEAN = false
Boolean_1 = true
switcher= on
switcher_1= off
switcher_2 = OFF
switcher_3 = ON
switcher_4 = Y
switcher_5 = N
flag = 1
[Dev:common]
string_1.1 = "testing_dev"
string_2.2 = "testing_dev"
CaseInsensitive = true
`
	)
	var iniConf interface{}
	err := ini.Unmarshal([]byte(iniContext), &iniConf)
	c.Assert(err, IsNil)
    value, ok := iniConf.(map[interface{}]interface{})
	c.Assert(ok, Equals, true, Commentf("value: %#v", iniConf))
    section_value, ok :=  value["common"].(map[interface{}]interface{})
    c.Assert(ok, Equals, true, Commentf("value: %#v", value["common"]))
    string_1_value, ok :=  section_value["string_1"].(map[interface{}]interface{})
    c.Assert(ok, Equals, true, Commentf("value: %#v", section_value["string_1"]))
    c.Assert(string_1_value["1"], DeepEquals, "testing")
	string_3_value, ok :=  section_value["string_3"].(map[interface{}]interface{})
	c.Assert(ok, Equals, true, Commentf("value: %#v", section_value["string_3"]))
	string_3_1_value, ok :=  string_3_value["1"].(map[interface{}]interface{})
	c.Assert(ok, Equals, true, Commentf("value: %#v", string_3_value["1"]))
    c.Assert(string_3_1_value["1"], DeepEquals, "testing_1")
	c.Assert(string_3_1_value["2"], DeepEquals, "testing_2")

	//buf, err := Marshal(reflect.ValueOf(iniConf))
	//fmt.Println(buf)
}
