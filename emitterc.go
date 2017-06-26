package ini

import ()

// Flush the buffer if needed.
func flush(emitter *ini_emitter_t) bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) {
		return ini_emitter_flush(emitter)
	}
	return true
}

// Put a character to the output buffer.
func put(emitter *ini_emitter_t, value byte) bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) && !ini_emitter_flush(emitter) {
		return false
	}
	emitter.buffer[emitter.buffer_pos] = value
	emitter.buffer_pos++
	emitter.column++
	return true
}

// Put a line break to the output buffer.
func put_break(emitter *ini_emitter_t) bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) && !ini_emitter_flush(emitter) {
		return false
	}
	switch emitter.line_break {
	case ini_CR_BREAK:
		emitter.buffer[emitter.buffer_pos] = '\r'
		emitter.buffer_pos += 1
	case ini_LN_BREAK:
		emitter.buffer[emitter.buffer_pos] = '\n'
		emitter.buffer_pos += 1
	case ini_CRLN_BREAK:
		emitter.buffer[emitter.buffer_pos+0] = '\r'
		emitter.buffer[emitter.buffer_pos+1] = '\n'
		emitter.buffer_pos += 2
	default:
		panic("unknown line break setting")
	}
	emitter.column = 0
	emitter.line++
	return true
}

// Copy a character from a string into buffer.
func write(emitter *ini_emitter_t, s []byte, i *int) bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) && !ini_emitter_flush(emitter) {
		return false
	}
	p := emitter.buffer_pos
	w := width(s[*i])
	switch w {
	case 4:
		emitter.buffer[p+3] = s[*i+3]
		fallthrough
	case 3:
		emitter.buffer[p+2] = s[*i+2]
		fallthrough
	case 2:
		emitter.buffer[p+1] = s[*i+1]
		fallthrough
	case 1:
		emitter.buffer[p+0] = s[*i+0]
	default:
		panic("unknown character width")
	}
	emitter.column++
	emitter.buffer_pos += w
	*i += w
	return true
}

// Write a whole string into buffer.
func write_all(emitter *ini_emitter_t, s []byte) bool {
	for i := 0; i < len(s); {
		if !write(emitter, s, &i) {
			return false
		}
	}
	return true
}

// Copy a line break character from a string into buffer.
func write_break(emitter *ini_emitter_t, s []byte, i *int) bool {
	if s[*i] == '\n' {
		if !put_break(emitter) {
			return false
		}
		*i++
	} else {
		if !write(emitter, s, i) {
			return false
		}
		emitter.column = 0
		emitter.line++
	}
	return true
}

// Set an emitter error and return false.
func ini_emitter_set_emitter_error(emitter *ini_emitter_t, problem string) bool {
	emitter.error = ini_EMITTER_ERROR
	emitter.problem = problem
	return false
}

// Emit an event.
func ini_emitter_emit(emitter *ini_emitter_t, event *ini_event_t) bool {
	emitter.events = append(emitter.events, *event)
	for !ini_emitter_need_more_events(emitter) {
		event := &emitter.events[emitter.events_head]
		if !ini_emitter_state_machine(emitter, event) {
			return false
		}
		ini_event_delete(event)
		emitter.events_head++
	}
	return true
}

// Check if we need to accumulate more events before emitting.
//
// We accumulate extra
//  - 1 event for DOCUMENT-START
//  - 2 events for SEQUENCE-START
//  - 3 events for MAPPING-START
//
func ini_emitter_need_more_events(emitter *ini_emitter_t) bool {
	if emitter.events_head == len(emitter.events) {
		return true
	}
	var accumulate int
	switch emitter.events[emitter.events_head].typ {
	case ini_DOCUMENT_START_EVENT:
		accumulate = 1
		break
	case ini_MAPPING_EVENT:
		accumulate = 2
		break
	case ini_COMMENT_EVENT:
		accumulate = 3
		break
	default:
		return false
	}
	if len(emitter.events)-emitter.events_head > accumulate {
		return false
	}
	var level int
	for i := emitter.events_head; i < len(emitter.events); i++ {
		switch emitter.events[i].typ {
		case ini_DOCUMENT_START_EVENT, ini_MAPPING_EVENT:
			level++
		}
		if level == 0 {
			return false
		}
	}
	return true
}

// State dispatcher.
func ini_emitter_state_machine(emitter *ini_emitter_t, event *ini_event_t) bool {
	switch emitter.state {
	default:
	case ini_EMIT_DOCUMENT_START_STATE:
		return ini_emitter_emit_start(emitter, event)
	case ini_EMIT_DOCUMENT_END_STATE:
		return ini_emitter_emit_document(emitter, event)
	case ini_EMIT_FIRST_SECTION_START_STATE:
		return ini_emitter_emit_section_start(emitter, event, true)
	case ini_EMIT_SECTION_START_STATE:
		return ini_emitter_emit_section_start(emitter, event, false)
	case ini_EMIT_SECTION_FIRST_NODE_KEY_STATE:
		return ini_emitter_emit_node(emitter, event)
	case ini_EMIT_ELEMENT_KEY_STATE:
		return ini_emitter_emit_node(emitter, event)
	case ini_EMIT_ELEMENT_VALUE_STATE:
		return ini_emitter_emit_node(emitter, event)
	case ini_EMIT_SECTION_END_STATE:
		return ini_emitter_emit_section_end(emitter, event, false)
	case ini_EMIT_COMMENT_START_STATE:
		return ini_emitter_emit_comment(emitter, event)
	}
	panic("invalid emitter state")
}

// Expect DOCUMENT-START.
func ini_emitter_emit_start(emitter *ini_emitter_t, event *ini_event_t) bool {
	if event.typ != ini_DOCUMENT_START_EVENT {
		return ini_emitter_set_emitter_error(emitter, "expected START")
	}
	if emitter.line_break == ini_ANY_BREAK {
		emitter.line_break = ini_LN_BREAK
	}

	emitter.line = 0
	emitter.column = 0
	emitter.whitespace = true

	emitter.state = ini_EMIT_DOCUMENT_END_STATE
	return true
}

func ini_emitter_emit_document(emitter *ini_emitter_t, event *ini_event_t) bool {
	if event.typ != ini_DOCUMENT_START_EVENT {
		return ini_emitter_set_emitter_error(emitter, "expected DOCUMENT-START")
	}
	if emitter.line_break == ini_ANY_BREAK {
		emitter.line_break = ini_LN_BREAK
	}

	emitter.line = 0
	emitter.whitespace = true

	emitter.state = ini_EMIT_FIRST_SECTION_START_STATE
	return true
}

// Expect a section start.
func ini_emitter_emit_section_start(emitter *ini_emitter_t, event *ini_event_t, first bool) bool {
	if first == true {
		if string(emitter.scalar_data.value) == DEFAULT_SECTION {

		}
	} else {
		if !ini_emitter_write_indicator(emitter, []byte{'['}, false, false) {
			return false
		}
		if !ini_emitter_process_element(emitter) {
			return false
		}
		if !ini_emitter_write_indicator(emitter, []byte{']'}, false, false) {
			return false
		}
	}
	emitter.states = append(emitter.states, ini_EMIT_SECTION_START_STATE)
	return ini_emitter_emit_node(emitter, event)
}

// Expect a section end.
func ini_emitter_emit_section_end(emitter *ini_emitter_t, event *ini_event_t, first bool) bool {
	emitter.states = append(emitter.states, ini_EMIT_SECTION_END_STATE)
	return ini_emitter_emit_node(emitter, event)
}

// Expect a node.
func ini_emitter_emit_node(emitter *ini_emitter_t, event *ini_event_t) bool {
	switch event.typ {
	case ini_SCALAR_EVENT:
		return ini_emitter_emit_scalar(emitter, event)
	case ini_COMMENT_EVENT:
		return ini_emitter_emit_comment(emitter, event)
	default:
		return ini_emitter_set_emitter_error(emitter,
			"expected SCALAR, SECTION-START, COMMENT-START")
	}
	return false
}

// Expect a comment.
func ini_emitter_emit_comment(emitter *ini_emitter_t, event *ini_event_t) bool {
	if !ini_emitter_write_indicator(emitter, []byte{'#'}, false, false) {
		return false
	}
	if !ini_emitter_process_element(emitter) {
		return false
	}
	return false
}

// Expect SCALAR.
func ini_emitter_emit_scalar(emitter *ini_emitter_t, event *ini_event_t) bool {
	if !ini_emitter_select_scalar_style(emitter, event) {
		return false
	}
	if !ini_emitter_process_element(emitter) {
		return false
	}
	emitter.state = emitter.states[len(emitter.states)-1]
	emitter.states = emitter.states[:len(emitter.states)-1]
	return true
}

// Check if the document content is an empty scalar.
func ini_emitter_check_empty_document(emitter *ini_emitter_t) bool {
	return false // [Go] Huh?
}

// Check if the next events represent an empty section.
func ini_emitter_check_empty_section(emitter *ini_emitter_t) bool {
	if len(emitter.events)-emitter.events_head < 2 {
		return false
	}
	if emitter.events[emitter.events_head].typ == ini_MAPPING_EVENT &&
			(emitter.events[emitter.events_head+1].typ == ini_MAPPING_EVENT ||  emitter.events[emitter.events_head+2].typ == ini_MAPPING_EVENT) {
		return true
	}
	return false

}

// Check if the next events represent an empty comment.
func ini_emitter_check_empty_comment(emitter *ini_emitter_t) bool {
	if len(emitter.events)-emitter.events_head < 2 {
		return false
	}
	return emitter.events[emitter.events_head].typ == ini_COMMENT_EVENT
}

// Determine an acceptable scalar style.
func ini_emitter_select_scalar_style(emitter *ini_emitter_t, event *ini_event_t) bool {

	style := event.scalar_style()
	if style == ini_ANY_SCALAR_STYLE {
		style = ini_PLAIN_SCALAR_STYLE
	}

	if style == ini_PLAIN_SCALAR_STYLE {
		if len(emitter.scalar_data.value) == 0 {
			style = ini_SINGLE_QUOTED_SCALAR_STYLE
		}
	}
	if style == ini_SINGLE_QUOTED_SCALAR_STYLE {
		if !emitter.scalar_data.single_quoted_allowed {
			style = ini_DOUBLE_QUOTED_SCALAR_STYLE
		}
	}

	emitter.scalar_data.style = style
	return true
}

// Write a scalar.
func ini_emitter_process_element(emitter *ini_emitter_t) bool {
	switch emitter.scalar_data.style {
	case ini_SINGLE_QUOTED_SCALAR_STYLE:
		return ini_emitter_write_single_quoted_element(emitter, emitter.scalar_data.value)

	case ini_DOUBLE_QUOTED_SCALAR_STYLE:
		return ini_emitter_write_double_quoted_element(emitter, emitter.scalar_data.value)
	}
	panic("unknown scalar style")
}

// Write the BOM character.
func ini_emitter_write_bom(emitter *ini_emitter_t) bool {
	if !flush(emitter) {
		return false
	}
	pos := emitter.buffer_pos
	emitter.buffer[pos+0] = '\xEF'
	emitter.buffer[pos+1] = '\xBB'
	emitter.buffer[pos+2] = '\xBF'
	emitter.buffer_pos += 3
	return true
}

func ini_emitter_write_indicator(emitter *ini_emitter_t, indicator []byte, need_whitespace, is_whitespace bool) bool {
	if need_whitespace && !emitter.whitespace {
		if !put(emitter, ' ') {
			return false
		}
	}
	if !write_all(emitter, indicator) {
		return false
	}
	emitter.whitespace = is_whitespace
	emitter.open_ended = false
	return true
}

func ini_emitter_write_single_quoted_element(emitter *ini_emitter_t, value []byte) bool {

	if !ini_emitter_write_indicator(emitter, []byte{'\''}, true, false) {
		return false
	}

	breaks := false
	for i := 0; i < len(value); {
		if is_space(value, i) {
			if !write(emitter, value, &i) {
				return false
			}
		} else if is_break(value, i) {
			if !breaks && value[i] == '\n' {
				if !put_break(emitter) {
					return false
				}
			}
			if !write_break(emitter, value, &i) {
				return false
			}
			breaks = true
		} else {
			if value[i] == '\'' {
				if !put(emitter, '\'') {
					return false
				}
			}
			if !write(emitter, value, &i) {
				return false
			}
			breaks = false
		}
	}
	if !ini_emitter_write_indicator(emitter, []byte{'\''}, false, false) {
		return false
	}
	emitter.whitespace = false
	return true
}

func ini_emitter_write_double_quoted_element(emitter *ini_emitter_t, value []byte) bool {
	if !ini_emitter_write_indicator(emitter, []byte{'"'}, true, false) {
		return false
	}

	for i := 0; i < len(value); {
		if !is_printable(value, i) || (!emitter.unicode && !is_ascii(value, i)) ||
			is_bom(value, i) || is_break(value, i) ||
			value[i] == '"' || value[i] == '\\' {

			octet := value[i]

			var w int
			var v rune
			switch {
			case octet&0x80 == 0x00:
				w, v = 1, rune(octet&0x7F)
			case octet&0xE0 == 0xC0:
				w, v = 2, rune(octet&0x1F)
			case octet&0xF0 == 0xE0:
				w, v = 3, rune(octet&0x0F)
			case octet&0xF8 == 0xF0:
				w, v = 4, rune(octet&0x07)
			}
			for k := 1; k < w; k++ {
				octet = value[i+k]
				v = (v << 6) + (rune(octet) & 0x3F)
			}
			i += w

			if !put(emitter, '\\') {
				return false
			}

			var ok bool
			switch v {
			case 0x00:
				ok = put(emitter, '0')
			case 0x07:
				ok = put(emitter, 'a')
			case 0x08:
				ok = put(emitter, 'b')
			case 0x09:
				ok = put(emitter, 't')
			case 0x0A:
				ok = put(emitter, 'n')
			case 0x0b:
				ok = put(emitter, 'v')
			case 0x0c:
				ok = put(emitter, 'f')
			case 0x0d:
				ok = put(emitter, 'r')
			case 0x1b:
				ok = put(emitter, 'e')
			case 0x22:
				ok = put(emitter, '"')
			case 0x5c:
				ok = put(emitter, '\\')
			case 0x85:
				ok = put(emitter, 'N')
			case 0xA0:
				ok = put(emitter, '_')
			case 0x2028:
				ok = put(emitter, 'L')
			case 0x2029:
				ok = put(emitter, 'P')
			default:
				if v <= 0xFF {
					ok = put(emitter, 'x')
					w = 2
				} else if v <= 0xFFFF {
					ok = put(emitter, 'u')
					w = 4
				} else {
					ok = put(emitter, 'U')
					w = 8
				}
				for k := (w - 1) * 4; ok && k >= 0; k -= 4 {
					digit := byte((v >> uint(k)) & 0x0F)
					if digit < 10 {
						ok = put(emitter, digit+'0')
					} else {
						ok = put(emitter, digit+'A'-10)
					}
				}
			}
			if !ok {
				return false
			}
		} else if is_space(value, i) {
			if !write(emitter, value, &i) {
				return false
			}
		} else {
			if !write(emitter, value, &i) {
				return false
			}
		}
	}
	if !ini_emitter_write_indicator(emitter, []byte{'"'}, false, false) {
		return false
	}
	emitter.whitespace = false
	return true
}
