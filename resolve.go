package ini

import (
	"encoding/base64"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type resolveMapItem struct {
	value interface{}
	tag   string
}

var resolveTable = make([]byte, 256)
var resolveMap = make(map[string]resolveMapItem)

func init() {
	t := resolveTable
	t[int('+')] = 'S' // Sign
	t[int('-')] = 'S'
	for _, c := range "0123456789" {
		t[int(c)] = 'D' // Digit
	}
    for _, c := range "yYnNtTfFoO~" {
        t[int(c)] = 'M' // In map
    }
	t[int('.')] = '.' // Float (potentially in map)

	var resolveMapList = []struct {
		v   interface{}
		tag string
		l   []string
	}{
		{true, ini_BOOL_TAG, []string{"y", "Y", "yes", "Yes", "YES"}},
		{true, ini_BOOL_TAG, []string{"true", "True", "TRUE"}},
		{true, ini_BOOL_TAG, []string{"on", "On", "ON"}},
		{false, ini_BOOL_TAG, []string{"n", "N", "no", "No", "NO"}},
		{false, ini_BOOL_TAG, []string{"false", "False", "FALSE"}},
		{false, ini_BOOL_TAG, []string{"off", "Off", "OFF"}},
		{nil, ini_NULL_TAG, []string{"", "~", "null", "Null", "NULL"}},
		{math.NaN(), ini_FLOAT_TAG, []string{".nan", ".NaN", ".NAN"}},
		{math.Inf(+1), ini_FLOAT_TAG, []string{".inf", ".Inf", ".INF"}},
		{math.Inf(+1), ini_FLOAT_TAG, []string{"+.inf", "+.Inf", "+.INF"}},
		{math.Inf(-1), ini_FLOAT_TAG, []string{"-.inf", "-.Inf", "-.INF"}},
	}

	m := resolveMap
	for _, item := range resolveMapList {
		for _, s := range item.l {
			m[s] = resolveMapItem{item.v, item.tag}
		}
	}
}

func resolvableTag(tag string) bool {
	switch tag {
	case "", ini_STR_TAG, ini_BOOL_TAG, ini_INT_TAG, ini_FLOAT_TAG, ini_NULL_TAG:
		return true
	}
	return false
}

var iniStyleFloat = regexp.MustCompile(`^[-+]?[0-9]*\.?[0-9]+([eE][-+][0-9]+)?$`)

func resolve(tag string, in string) (rtag string, out interface{}) {
	if !resolvableTag(tag) {
		return tag, in
	}

	defer func() {
		switch tag {
		case "", rtag, ini_STR_TAG, ini_BINARY_TAG:
			return
		}
		failf("cannot decode %s `%s` as a %s", rtag, in, tag)
	}()

	// Any data is accepted as a !!str or !!binary.
	// Otherwise, the prefix is enough of a hint about what it might be.
	hint := byte('N')
	if in != "" {
		hint = resolveTable[in[0]]
	}
	if hint != 0 && tag != ini_STR_TAG && tag != ini_BINARY_TAG {
		// Handle things we can lookup in a map.
		if item, ok := resolveMap[in]; ok {
			return item.tag, item.value
		}

		// Base 60 floats are a bad idea, were dropped in YAML 1.2, and
		// are purposefully unsupported here. They're still quoted on
		// the way out for compatibility with other parser, though.

		switch hint {
		case 'M':
			// We've already checked the map above.

		case '.':
			// Not in the map, so maybe a normal float.
			floatv, err := strconv.ParseFloat(in, 64)
			if err == nil {
				return ini_FLOAT_TAG, floatv
			}

		case 'D', 'S':
			// Int, float, or timestamp.
			plain := strings.Replace(in, "_", "", -1)
			intv, err := strconv.ParseInt(plain, 0, 64)
			if err == nil {
				if intv == int64(int(intv)) {
					return ini_INT_TAG, int(intv)
				} else {
					return ini_INT_TAG, intv
				}
			}
			uintv, err := strconv.ParseUint(plain, 0, 64)
			if err == nil {
				return ini_INT_TAG, uintv
			}
			if iniStyleFloat.MatchString(plain) {
				floatv, err := strconv.ParseFloat(plain, 64)
				if err == nil {
					return ini_FLOAT_TAG, floatv
				}
			}
			if strings.HasPrefix(plain, "0b") {
				intv, err := strconv.ParseInt(plain[2:], 2, 64)
				if err == nil {
					if intv == int64(int(intv)) {
						return ini_INT_TAG, int(intv)
					} else {
						return ini_INT_TAG, intv
					}
				}
				uintv, err := strconv.ParseUint(plain[2:], 2, 64)
				if err == nil {
					return ini_INT_TAG, uintv
				}
			} else if strings.HasPrefix(plain, "-0b") {
				intv, err := strconv.ParseInt(plain[3:], 2, 64)
				if err == nil {
					if intv == int64(int(intv)) {
						return ini_INT_TAG, -int(intv)
					} else {
						return ini_INT_TAG, -intv
					}
				}
			}
			// XXX Handle timestamps here.

		default:
			panic("resolveTable item not yet handled: " + string(rune(hint)) + " (with " + in + ")")
		}
	}
	if tag == ini_BINARY_TAG {
		return ini_BINARY_TAG, in
	}
	if utf8.ValidString(in) {
		return ini_STR_TAG, in
	}
	return ini_BINARY_TAG, encodeBase64(in)
}

// encodeBase64 encodes s as base64 that is broken up into multiple lines
// as appropriate for the resulting length.
func encodeBase64(s string) string {
	const lineLen = 70
	encLen := base64.StdEncoding.EncodedLen(len(s))
	lines := encLen/lineLen + 1
	buf := make([]byte, encLen*2+lines)
	in := buf[0:encLen]
	out := buf[encLen:]
	base64.StdEncoding.Encode(in, []byte(s))
	k := 0
	for i := 0; i < len(in); i += lineLen {
		j := i + lineLen
		if j > len(in) {
			j = len(in)
		}
		k += copy(out[k:], in[i:j])
		if lines > 1 {
			out[k] = '\n'
			k++
		}
	}
	return string(out[:k])
}
