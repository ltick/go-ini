package ini

import (
	"encoding"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type encoder struct {
	emitter ini_emitter_t
	event   ini_event_t
	out     []byte
	flow    bool
}

func newEncoder() (e *encoder) {
	e = &encoder{}
	e.must(ini_emitter_initialize(&e.emitter))
	ini_emitter_set_output_string(&e.emitter, &e.out)
	ini_emitter_set_unicode(&e.emitter, true)
	e.must(ini_document_start_event_initialize(&e.event))
	e.emit()
	return e
}

func (e *encoder) finish() {
	e.must(ini_document_start_event_initialize(&e.event))
	e.emit()
	e.emitter.open_ended = false
}

func (e *encoder) destroy() {
	ini_emitter_delete(&e.emitter)
}

func (e *encoder) emit() {
	// This will internally delete the e.event value.
	if !ini_emitter_emit(&e.emitter, &e.event) && e.event.typ != ini_DOCUMENT_END_EVENT {
		e.must(false)
	}
}

func (e *encoder) must(ok bool) {
	if !ok {
		msg := e.emitter.problem
		if msg == "" {
			msg = "unknown problem generating INI content"
		}
		failf("%s", msg)
	}
}

func (e *encoder) marshal(in reflect.Value) {
	if !in.IsValid() {
		e.nilv()
		return
	}
	iface := in.Interface()
	if m, ok := iface.(Marshaler); ok {
		v, err := m.MarshalINI()
		if err != nil {
			fail(err)
		}
		if v == nil {
			e.nilv()
			return
		}
		in = reflect.ValueOf(v)
	} else if m, ok := iface.(encoding.TextMarshaler); ok {
		text, err := m.MarshalText()
		if err != nil {
			fail(err)
		}
		in = reflect.ValueOf(string(text))
	}
	switch in.Kind() {
	case reflect.Interface:
		if in.IsNil() {
			e.nilv()
		} else {
			e.marshal(in.Elem())
		}
	case reflect.Map:
		e.mapv(in)
	case reflect.Ptr:
		if in.IsNil() {
			e.nilv()
		} else {
			e.marshal(in.Elem())
		}
	case reflect.String:
		e.stringv(in)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if in.Type() == durationType {
			e.stringv(reflect.ValueOf(iface.(time.Duration).String()))
		} else {
			e.intv(in)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		e.uintv(in)
	case reflect.Float32, reflect.Float64:
		e.floatv(in)
	case reflect.Bool:
		e.boolv(in)
	default:
		panic("cannot marshal type: " + in.Type().String())
	}
}

func (e *encoder) mapv(in reflect.Value) {
}

// isBase60 returns whether s is in base 60 notation as defined in YAML 1.1.
//
// The base 60 float notation in YAML 1.1 is a terrible idea and is unsupported
// in YAML 1.2 and by this package, but these should be marshalled quoted for
// the time being for compatibility with other parsers.
func isBase60Float(s string) (result bool) {
	// Fast path.
	if s == "" {
		return false
	}
	c := s[0]
	if !(c == '+' || c == '-' || c >= '0' && c <= '9') || strings.IndexByte(s, ':') < 0 {
		return false
	}
	// Do the full match.
	return base60float.MatchString(s)
}

// From http://ini.org/type/float.html, except the regular expression there
// is bogus. In practice parsers do not enforce the "\.[0-9_]*" suffix.
var base60float = regexp.MustCompile(`^[-+]?[0-9][0-9_]*(?::[0-5]?[0-9])+(?:\.[0-9_]*)?$`)

func (e *encoder) stringv(in reflect.Value) {
	var style ini_scalar_style_t
	s := in.String()
	if isBase60Float(s) {
		style = ini_DOUBLE_QUOTED_SCALAR_STYLE
	} else if strings.Contains(s, "\n") {
		style = ini_PLAIN_SCALAR_STYLE
	} else {
		style = ini_PLAIN_SCALAR_STYLE
	}
	e.emitNode(s, style)
}

func (e *encoder) boolv(in reflect.Value) {
	var s string
	if in.Bool() {
		s = "true"
	} else {
		s = "false"
	}
	e.emitNode(s, ini_PLAIN_SCALAR_STYLE)
}

func (e *encoder) intv(in reflect.Value) {
	s := strconv.FormatInt(in.Int(), 10)
	e.emitNode(s, ini_PLAIN_SCALAR_STYLE)
}

func (e *encoder) uintv(in reflect.Value) {
	s := strconv.FormatUint(in.Uint(), 10)
	e.emitNode(s, ini_PLAIN_SCALAR_STYLE)
}

func (e *encoder) floatv(in reflect.Value) {
	// FIXME: Handle 64 bits here.
	s := strconv.FormatFloat(float64(in.Float()), 'g', -1, 32)
	switch s {
	case "+Inf":
		s = ".inf"
	case "-Inf":
		s = "-.inf"
	case "NaN":
		s = ".nan"
	}
	e.emitNode(s, ini_PLAIN_SCALAR_STYLE)
}

func (e *encoder) nilv() {
	e.emitNode("null", ini_PLAIN_SCALAR_STYLE)
}

func (e *encoder) emitNode(value string, style ini_scalar_style_t) {
	e.must(ini_scalar_event_initialize(&e.event, []byte(value), style))
	e.emit()
}
