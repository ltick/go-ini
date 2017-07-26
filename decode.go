package ini

import (
	"encoding"
	"encoding/base64"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
)

const (
	documentNode = 1 << iota
	inheritNode
	sectionNode
	mappingNode
	scalarNode
	commentNode
)

type node struct {
	kind         int
	line, column int
	tag          string
	value        string
	children     []*node
}

// ----------------------------------------------------------------------------
// Parser, produces a node tree out of a ini document.

type parser struct {
	parser ini_parser_t
	event  ini_event_t
	doc    *node
}

func newParser(b []byte) *parser {
	p := parser{}
	if !ini_parser_initialize(&p.parser) {
		panic("failed to initialize INI parser")
	}

	if len(b) == 0 {
		b = []byte{}
	}

	ini_parser_set_input_string(&p.parser, b)

	p.skip()
	if p.event.typ != ini_DOCUMENT_START_EVENT {
		panic("expected ini_DOCUMENT_START_EVENT, got " + p.event.event_type())
	}
	return &p
}

func (p *parser) destroy() {
	if p.event.typ != ini_NO_EVENT {
		ini_event_delete(&p.event)
	}
	ini_parser_delete(&p.parser)
}

func (p *parser) skip() {
	if p.event.typ != ini_NO_EVENT {
		if p.event.typ == ini_DOCUMENT_END_EVENT {
			failf("attempted to go past the end of document; corrupted value?")
		}
		ini_event_delete(&p.event)
	}
	if !ini_parser_parse(&p.parser, &p.event) {
		p.fail()
	}
}

func (p *parser) fail() {
	var where string
	var line int
	if p.parser.problem_mark.line != 0 {
		line = p.parser.problem_mark.line
	} else if p.parser.context_mark.line != 0 {
		line = p.parser.context_mark.line
	}
	if line != 0 {
		where = "line " + strconv.Itoa(line) + ": "
	}
	var msg string
	if len(p.parser.problem) > 0 {
		msg = p.parser.problem
	} else {
		msg = "unknown problem parsing INI content"
	}
	failf("%s%s", where, msg)
}

func (p *parser) parse() *node {
	switch p.event.typ {
	case ini_DOCUMENT_START_EVENT:
		return p.document()
	case ini_SECTION_INHERIT_EVENT:
		return p.inherit()
	case ini_SECTION_ENTRY_EVENT:
		return p.section()
	case ini_MAPPING_EVENT:
		return p.mapping()
	case ini_SCALAR_EVENT:
		return p.scalar()
	case ini_COMMENT_EVENT:
		return p.comment()
	case ini_DOCUMENT_END_EVENT:
		// Happens when attempting to decode an empty buffer.
		return nil
	default:
		panic("attempted to parse unknown event: " + p.event.event_type())
	}
}

func (p *parser) node(kind int) *node {
	return &node{
		kind:   kind,
		line:   p.event.start_mark.line,
		column: p.event.start_mark.column,
	}
}

func (p *parser) clone_node(n *node) *node {
	thisNode := p.node(n.kind)
	thisNode.tag = n.tag
	thisNode.value = n.value
	for _, childNode := range n.children {
		thisNode.children = append(thisNode.children, p.clone_node(childNode))
	}
	return thisNode
}


/**
targetNode is: "hello": [1: "world"]
sourceNode is: "hello": [1: [2: "world"]]
overwrite result is:
"hello": [1: [2: "world"]]
no overwrite result is:
"hello": [1: "world"]

Specific scene:
when inherit a section , the expect operation for the same node is "no overwrite"
in the same section, the expect operation for the same node is "overwrite"
*/
func (p *parser) merge_node(targetNode *node, sourceNode *node, overwrite bool) {
	if targetNode.kind == sourceNode.kind {
		targetNodeCount := len(targetNode.children)
		sourceNodeCount := len(sourceNode.children)
		swapChildNodes := make([]*node, 0)
		for i := 0; i < sourceNodeCount; i += 2 {
			nodeExist := false
			for j := 0; j < targetNodeCount; j += 2 {
				if sourceNode.children[i].kind == scalarNode && targetNode.children[j].kind == scalarNode && sourceNode.children[i].value == targetNode.children[j].value {
					nodeExist = true
					if sourceNode.children[i+1].kind == targetNode.children[j+1].kind {
						if len(sourceNode.children[i+1].children) > 0 && len(targetNode.children[j+1].children) > 0 {
							p.merge_node(targetNode.children[j+1], p.clone_node(sourceNode.children[i+1]), overwrite)
						}
					} else {
                        if overwrite {
                            targetNode.children[j + 1] = p.clone_node(sourceNode.children[i + 1])
                        }
					}
					break
				}
			}
			if !nodeExist {
				swapChildNodes = append(swapChildNodes, sourceNode.children[i], sourceNode.children[i+1])
			}
		}
		swapChildNodeCount := len(swapChildNodes)
		for i := 0; i < swapChildNodeCount; i += 2 {
			targetNode.children = append(targetNode.children, swapChildNodes[i], swapChildNodes[i+1])
		}
	}
	return
}

func (p *parser) document() *node {
	n := p.node(documentNode)
	p.doc = n
	p.skip()
	for p.event.typ != ini_DOCUMENT_END_EVENT {
		keyNode := p.parse()
		nextNode := p.parse()
		if nextNode.kind == inheritNode {
			childNode := p.parse()
			// inherit
			sectionExists := false
			for i := 0; i < len(p.doc.children); i += 2 {
				if p.doc.children[i].kind == scalarNode && p.doc.children[i].value == nextNode.value {
					sectionExists = true
					p.merge_node(childNode, p.clone_node(p.doc.children[i+1]), false)
					break
				}
			}
			if !sectionExists && nextNode.value != DEFAULT_SECTION {
				failf("inherit section '%s' does not exists", nextNode.value)
			}
			n.children = append(n.children, keyNode, childNode)
		} else if nextNode.kind == sectionNode {
			n.children = append(n.children, keyNode, nextNode)
		}
		p.skip()
	}
	return n
}

func (p *parser) section() *node {
	thisNode := p.node(sectionNode)

	// until next ini_SECTION_START_EVENT
	p.skip()
	parentNode := thisNode
	for p.event.typ != ini_SECTION_ENTRY_EVENT {
		currentNodeKey := p.parse()
		if currentNodeKey == nil {
			p.fail()
		}
		if currentNodeKey.kind == scalarNode {
			currentNodeValue := p.parse()
			swapChildNodes := make([]*node, 0)
			for i := 0; i < len(parentNode.children); i += 2 {
				if parentNode.children[i].value == currentNodeKey.value {
					if parentNode.children[i+1].kind == currentNodeValue.kind {
						swapChildNodes = append(swapChildNodes, parentNode.children[i], parentNode.children[i+1])
					}
				} else {
					swapChildNodes = append(swapChildNodes, parentNode.children[i], parentNode.children[i+1])
				}
			}
			parentNode.children = swapChildNodes

			nodeExist := false
			for i := 0; i < len(parentNode.children); i += 2 {
				// condition:
				// 1. current node type
				// 2. current node value
				if currentNodeKey.kind == scalarNode && parentNode.children[i].kind == scalarNode && currentNodeKey.value == parentNode.children[i].value {
					nodeExist = true
					// if current node value type is different, overwrite it
					if parentNode.children[i+1].kind != currentNodeValue.kind {
						parentNode.children[i+1] = p.clone_node(currentNodeValue)
					} else {
						p.merge_node(parentNode.children[i+1], p.clone_node(currentNodeValue), true)
					}
					break
				}
			}
			if !nodeExist {
				parentNode.children = append(parentNode.children, currentNodeKey, currentNodeValue)
			}
		}
		parentNode = thisNode
	}
	return thisNode
}

func (p *parser) mapping() *node {
	thisNode := p.node(mappingNode)
	// until next ini_SECTION_START_EVENT
	p.skip()
	parentNode := thisNode
	currentNodeKey := p.parse()
	if currentNodeKey == nil {
		p.fail()
	}
	if currentNodeKey.kind == scalarNode {
		currentNodeValue := p.parse()
		nodeExist := false
		i := 0
		for ; i < len(parentNode.children); i += 2 {
			// condition:
			// 1. current node type
			// 2. current node value
			if currentNodeKey.kind == parentNode.children[i].kind && currentNodeKey.value == parentNode.children[i].value {
				nodeExist = true
				break
			}
		}
		if nodeExist {
			if len(parentNode.children) > 0 {
				// if node type is different, overwrite it
				if parentNode.children[i+1].kind != currentNodeValue.kind {
					parentNode.children[i+1] = p.clone_node(currentNodeValue)
				} else {
					p.merge_node(parentNode.children[i+1], p.clone_node(currentNodeValue), true)
				}
				parentNode = parentNode.children[i+1]
			}
		} else {
			parentNode.children = append(parentNode.children, currentNodeKey, currentNodeValue)
		}
		if currentNodeValue.kind == mappingNode {
			parentNode = currentNodeValue
		}
	}
	return thisNode
}

func (p *parser) inherit() *node {
	thisNode := p.node(inheritNode)
	thisNode.value = string(p.event.value)
	p.skip()
	return thisNode
}

func (p *parser) comment() *node {
	thisNode := p.node(commentNode)
	thisNode.value = string(p.event.value)
	p.skip()
	return thisNode
}

func (p *parser) scalar() *node {
	thisNode := p.node(scalarNode)
	thisNode.value = string(p.event.value)
	thisNode.tag = string(p.event.tag)
	p.skip()
	return thisNode
}

// ----------------------------------------------------------------------------
// Decoder, unmarshals a node into a provided value.

type decoder struct {
	doc     *node
	mapType reflect.Type
	terrors []string
}

var (
	mapItemType    = reflect.TypeOf(MapItem{})
	durationType   = reflect.TypeOf(time.Duration(0))
	defaultMapType = reflect.TypeOf(map[interface{}]interface{}{})
	ifaceType      = defaultMapType.Elem()
)

func newDecoder() *decoder {
	d := &decoder{mapType: defaultMapType}
	return d
}

func (d *decoder) terror(n *node, tag string, out reflect.Value) {
	if n.tag != "" {
		tag = n.tag
	}
	value := n.value
	if len(value) > 10 {
		value = " `" + value[:7] + "...`"
	} else {
		value = " `" + value + "`"
	}
	d.terrors = append(d.terrors, fmt.Sprintf("line %d: cannot unmarshal %s%s into %s", n.line+1, tag, value, out.Type()))
}

func (d *decoder) callUnmarshaler(n *node, u Unmarshaler) (good bool) {
	terrlen := len(d.terrors)
	err := u.UnmarshalINI(func(v interface{}) (err error) {
		defer handleErr(&err)
		d.unmarshal(n, reflect.ValueOf(v))
		if len(d.terrors) > terrlen {
			issues := d.terrors[terrlen:]
			d.terrors = d.terrors[:terrlen]
			return &TypeError{issues}
		}
		return nil
	})
	if e, ok := err.(*TypeError); ok {
		d.terrors = append(d.terrors, e.Errors...)
		return false
	}
	if err != nil {
		fail(err)
	}
	return true
}

// d.prepare initializes and dereferences pointers and calls UnmarshalINI
// if a value is found to implement it.
// It returns the initialized and dereferenced out value, whether
// unmarshalling was already done by UnmarshalINI, and if so whether
// its types unmarshalled appropriately.
//
// If n holds a null value, prepare returns before doing anything.
func (d *decoder) prepare(n *node, out reflect.Value) (newout reflect.Value, unmarshaled, good bool) {
	if n.tag == ini_NULL_TAG || n.kind == scalarNode && n.tag == "" && (n.value == "null" || n.value == "") {
		return out, false, false
	}
	again := true
	for again {
		again = false
		if out.Kind() == reflect.Ptr {
			if out.IsNil() {
				out.Set(reflect.New(out.Type().Elem()))
			}

			out = out.Elem()
			again = true
		}
		if out.CanAddr() {
			if u, ok := out.Addr().Interface().(Unmarshaler); ok {
				good = d.callUnmarshaler(n, u)
				return out, true, good
			}
		}
	}
	return out, false, false
}

func (d *decoder) unmarshal(n *node, out reflect.Value) (good bool) {
	out, unmarshaled, good := d.prepare(n, out)
	if unmarshaled {
		return good
	}
	switch n.kind {
	case documentNode:
		good = d.document(n, out)
	case sectionNode:
		good = d.mapping(n, out)
	case mappingNode:
		good = d.mapping(n, out)
	case scalarNode:
		good = d.scalar(n, out)
	default:
		panic("internal error: unknown node kind: " + strconv.Itoa(n.kind))
	}
	return good
}

func (d *decoder) document(n *node, out reflect.Value) (good bool) {
	if len(n.children) > 0 {
		d.doc = n
		switch out.Kind() {
		case reflect.Struct:
			sinfo, err := getStructInfo(out.Type())
			if err != nil {
				panic(err)
			}
			k := settableValueOf("")
			l := len(n.children)
			for i := 0; i < l; i += 2 {
				if !d.unmarshal(n.children[i], k) {
					continue
				}
				if n.children[i].value == DEFAULT_SECTION {
					ll := len(n.children[i+1].children)
					for j := 0; j < ll; j += 2 {
						if !d.unmarshal(n.children[i+1].children[j], k) {
							continue
						}
						if info, ok := sinfo.FieldsMap[k.String()]; ok {
							var field reflect.Value
							if info.Inline == nil {
								field = out.Field(info.Num)
							} else {
								field = out.FieldByIndex(info.Inline)
							}
							d.unmarshal(n.children[i+1].children[j+1], field)
						}
					}
				} else {
					if info, ok := sinfo.FieldsMap[k.String()]; ok {
						var field reflect.Value
						if info.Inline == nil {
							field = out.Field(info.Num)
						} else {
							field = out.FieldByIndex(info.Inline)
						}
						d.unmarshal(n.children[i+1], field)
					}
				}
			}
			return true
		case reflect.Slice:
			outt := out.Type()
			if outt.Elem() != mapItemType {
				d.terror(n, ini_MAP_TAG, out)
				return false
			}

			mapType := d.mapType
			d.mapType = outt

			var slice []MapItem
			var l = len(n.children)
			for i := 0; i < l; i += 2 {
				item := MapItem{}
				k := reflect.ValueOf(&item.Key).Elem()
				if !d.unmarshal(n.children[i], k) {
					continue
				}
				if n.children[i].value == DEFAULT_SECTION {
					ll := len(n.children[i+1].children)
					for j := 0; j < ll; j += 2 {
						item := MapItem{}
						k := reflect.ValueOf(&item.Key).Elem()
						if !d.unmarshal(n.children[j], k) {
							continue
						}
						v := reflect.ValueOf(&item.Value).Elem()
						if d.unmarshal(n.children[i+1], v) {
							slice = append(slice, item)
						}
					}
				} else {
					v := reflect.ValueOf(&item.Value).Elem()
					if d.unmarshal(n.children[i+1], v) {
						slice = append(slice, item)
					}
				}
			}
			out.Set(reflect.ValueOf(slice))
			d.mapType = mapType
			return true
		case reflect.Map:
			//okay
		case reflect.Interface:
			if d.mapType.Kind() == reflect.Map {
				iface := out
				out = reflect.MakeMap(d.mapType)
				iface.Set(out)
			} else {
				slicev := reflect.New(d.mapType).Elem()
				if !d.mappingSlice(n, slicev) {
					return false
				}
				out.Set(slicev)
				return true
			}
		default:
			d.terror(n, ini_MAP_TAG, out)
			return false
		}
		outt := out.Type()
		kt := outt.Key()
		et := outt.Elem()

		mapType := d.mapType
		if outt.Key() == ifaceType && outt.Elem() == ifaceType {
			d.mapType = outt
		}

		if out.IsNil() {
			out.Set(reflect.MakeMap(outt))
		}
		l := len(n.children)
		for i := 0; i < l; i += 2 {
			k := reflect.New(kt).Elem()
			if !d.unmarshal(n.children[i], k) {
				continue
			}
			kkind := k.Kind()
			if kkind == reflect.Interface {
				kkind = k.Elem().Kind()
			}
			if kkind == reflect.Map || kkind == reflect.Slice {
				failf("invalid map key: %#v", k.Interface())
			}
			e := reflect.New(et).Elem()
			if n.children[i].value == DEFAULT_SECTION {
				d.unmarshal(n.children[i+1], out)
			} else {
				if d.unmarshal(n.children[i+1], e) {
					out.SetMapIndex(k, e)
				}
			}
		}
		d.mapType = mapType
		return true
	}
	return false
}

var zeroValue reflect.Value

func resetMap(out reflect.Value) {
	for _, k := range out.MapKeys() {
		out.SetMapIndex(k, zeroValue)
	}
}

func settableValueOf(i interface{}) reflect.Value {
	v := reflect.ValueOf(i)
	sv := reflect.New(v.Type()).Elem()
	sv.Set(v)
	return sv
}

func (d *decoder) mapping(n *node, out reflect.Value) (good bool) {
	switch out.Kind() {
	case reflect.Struct:
		return d.mappingStruct(n, out)
	case reflect.Slice:
		return d.mappingSlice(n, out)
	case reflect.Map:
	// okay
	case reflect.Interface:
		if d.mapType.Kind() == reflect.Map {
			iface := out
			out = reflect.MakeMap(d.mapType)
			iface.Set(out)
		} else {
			slicev := reflect.New(d.mapType).Elem()
			if !d.mappingSlice(n, slicev) {
				return false
			}
			out.Set(slicev)
			return true
		}
	default:
		d.terror(n, ini_MAP_TAG, out)
		return false
	}
	outt := out.Type()
	kt := outt.Key()
	et := outt.Elem()

	mapType := d.mapType
	if outt.Key() == ifaceType && outt.Elem() == ifaceType {
		d.mapType = outt
	}

	if out.IsNil() {
		out.Set(reflect.MakeMap(outt))
	}
	l := len(n.children)
	for i := 0; i < l; i += 2 {
		k := reflect.New(kt).Elem()
		if d.unmarshal(n.children[i], k) {
			kkind := k.Kind()
			if kkind == reflect.Interface {
				kkind = k.Elem().Kind()
			}
			if kkind == reflect.Map || kkind == reflect.Slice {
				failf("invalid map key: %#v", k.Interface())
			}
			e := reflect.New(et).Elem()
			if d.unmarshal(n.children[i+1], e) {
				out.SetMapIndex(k, e)
			}
		}
	}
	d.mapType = mapType
	return true
}

func (d *decoder) mappingSlice(n *node, out reflect.Value) (good bool) {
	outt := out.Type()
	if outt.Elem() != mapItemType {
		d.terror(n, ini_MAP_TAG, out)
		return false
	}

	mapType := d.mapType
	d.mapType = outt

	var slice []MapItem
	var l = len(n.children)
	for i := 0; i < l; i += 2 {
		item := MapItem{}
		k := reflect.ValueOf(&item.Key).Elem()
		if d.unmarshal(n.children[i], k) {
			v := reflect.ValueOf(&item.Value).Elem()
			if d.unmarshal(n.children[i+1], v) {
				slice = append(slice, item)
			}
		}
	}
	out.Set(reflect.ValueOf(slice))
	d.mapType = mapType
	return true
}

func (d *decoder) mappingStruct(n *node, out reflect.Value) (good bool) {
	sinfo, err := getStructInfo(out.Type())
	if err != nil {
		panic(err)
	}
	name := settableValueOf("")
	l := len(n.children)
	for i := 0; i < l; i += 2 {
		if !d.unmarshal(n.children[i], name) {
			continue
		}
		if info, ok := sinfo.FieldsMap[name.String()]; ok {
			var field reflect.Value
			if info.Inline == nil {
				field = out.Field(info.Num)
			} else {
				field = out.FieldByIndex(info.Inline)
			}
			d.unmarshal(n.children[i+1], field)
		}
	}
	return true
}

func (d *decoder) scalar(n *node, out reflect.Value) (good bool) {
	var tag string
	var resolved interface{}

	tag, resolved = resolve(n.tag, n.value)
	if tag == ini_BINARY_TAG {
		data, err := base64.StdEncoding.DecodeString(resolved.(string))
		if err != nil {
			failf("!!binary value contains invalid base64 data")
		}
		resolved = string(data)
	}
	if resolved == nil {
		if out.Kind() == reflect.Map && !out.CanAddr() {
			resetMap(out)
		} else {
			out.Set(reflect.Zero(out.Type()))
		}
		return true
	}
	if s, ok := resolved.(string); ok && out.CanAddr() {
		if u, ok := out.Addr().Interface().(encoding.TextUnmarshaler); ok {
			err := u.UnmarshalText([]byte(s))
			if err != nil {
				fail(err)
			}
			return true
		}
	}
	switch out.Kind() {
	case reflect.String:
		if tag == ini_BINARY_TAG {
			out.SetString(resolved.(string))
			good = true
		} else if resolved != nil {
			out.SetString(n.value)
			good = true
		}
	case reflect.Interface:
		if resolved == nil {
			out.Set(reflect.Zero(out.Type()))
		} else {
			// TODO Not sure if we should resolve interface type of scalar
			switch resolved.(type) {
			//case bool:
			//	var resolvedString string = strconv.FormatBool(resolved.(bool))
			//	out.Set(reflect.ValueOf(resolvedString))
			//case string:
			//	var resolvedString string = resolved.(string)
			//	out.Set(reflect.ValueOf(resolvedString))
			//case int:
			//	var resolvedString string = strconv.FormatInt(int64(resolved.(int)), 10)
			//	out.Set(reflect.ValueOf(resolvedString))
			//case int64:
			//	var resolvedString string = strconv.FormatInt(resolved.(int64), 10)
			//	out.Set(reflect.ValueOf(resolvedString))
			//case uint64:
			//	var resolvedString string = strconv.FormatUint(resolved.(uint64), 10)
			//	out.Set(reflect.ValueOf(resolvedString))
			default:
				out.Set(reflect.ValueOf(resolved))
			}
		}
		good = true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch resolved := resolved.(type) {
		case int:
			if !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				good = true
			}
		case int64:
			if !out.OverflowInt(resolved) {
				out.SetInt(resolved)
				good = true
			}
		case uint64:
			if resolved <= math.MaxInt64 && !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				good = true
			}
		case float64:
			if resolved <= math.MaxInt64 && !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				good = true
			}
		case string:
			if out.Type() == durationType {
				d, err := time.ParseDuration(resolved)
				if err == nil {
					out.SetInt(int64(d))
					good = true
				}
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		switch resolved := resolved.(type) {
		case int:
			if resolved >= 0 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				good = true
			}
		case int64:
			if resolved >= 0 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				good = true
			}
		case uint64:
			if !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				good = true
			}
		case float64:
			if resolved <= math.MaxUint64 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				good = true
			}
		}
	case reflect.Bool:
		switch resolved := resolved.(type) {
		case bool:
			out.SetBool(resolved)
			good = true
		}
	case reflect.Float32, reflect.Float64:
		switch resolved := resolved.(type) {
		case int:
			out.SetFloat(float64(resolved))
			good = true
		case int64:
			out.SetFloat(float64(resolved))
			good = true
		case uint64:
			out.SetFloat(float64(resolved))
			good = true
		case float64:
			out.SetFloat(resolved)
			good = true
		}
	case reflect.Ptr:
		if out.Type().Elem() == reflect.TypeOf(resolved) {
			// TODO DOes this make sense? When is out a Ptr except when decoding a nil value?
			elem := reflect.New(out.Type().Elem())
			elem.Elem().Set(reflect.ValueOf(resolved))
			out.Set(elem)
			good = true
		}
	}
	if !good {
		d.terror(n, tag, out)
	}
	return good
}
