package ini

// The parser implements the following grammar:
//
// document		::= DOCUMENT-START section* DOCUMENT-END
// section      ::= SECTION-START (node | comment)* SECTION-END
// node         ::= KEY VALUE SCALAR
// comment      ::= COMMENT SCALAR

// Peek the next token in the token queue.
func peek_token(parser *ini_parser_t) *ini_token_t {
	if parser.token_available || ini_parser_fetch_more_tokens(parser) {
		return &parser.tokens[parser.tokens_head]
	}
	return nil
}

// Remove the next token from the queue (must be called after peek_token).
func skip_token(parser *ini_parser_t) {
	parser.token_available = false
	parser.tokens_parsed++
	parser.document_end_produced = parser.tokens[parser.tokens_head].typ == ini_DOCUMENT_END_TOKEN
	parser.tokens_head++
}

// Get the next event.
func ini_parser_parse(parser *ini_parser_t, event *ini_event_t) bool {
	// Erase the event object.
	*event = ini_event_t{}

	// No events after the end of the stream or error.
	if parser.document_end_produced || parser.error != ini_NO_ERROR || parser.state == ini_PARSE_DOCUMENT_END_STATE {
		return true
	}

	// Generate the next event.
	return ini_parser_state_machine(parser, event)
}

// Set parser error.
func ini_parser_set_parser_error(parser *ini_parser_t, problem string, problem_mark ini_mark_t) bool {
	parser.error = ini_PARSER_ERROR
	parser.problem = problem
	parser.problem_mark = problem_mark
	return false
}

func ini_parser_set_parser_error_context(parser *ini_parser_t, context string, context_mark ini_mark_t, problem string, problem_mark ini_mark_t) bool {
	parser.error = ini_PARSER_ERROR
	parser.context = context
	parser.context_mark = context_mark
	parser.problem = problem
	parser.problem_mark = problem_mark
	return false
}

// State dispatcher.
func ini_parser_state_machine(parser *ini_parser_t, event *ini_event_t) bool {
	//trace("ini_parser_state_machine", "state:", parser.state.String())
	switch parser.state {
	case ini_PARSE_DOCUMENT_START_STATE:
		return ini_parser_parse_document_start(parser, event)
	case ini_PARSE_SECTION_FIRST_START_STATE:
		return ini_parser_parse_section_start(parser, event, true)
	case ini_PARSE_SECTION_START_STATE:
		return ini_parser_parse_section_start(parser, event, false)
	case ini_PARSE_SECTION_INHERIT_STATE:
		return ini_parser_parse_section_inherit(parser, event)
	case ini_PARSE_SECTION_ENTRY_STATE:
		return ini_parser_parse_section_entry(parser, event)
	case ini_PARSE_SECTION_KEY_STATE:
		return ini_parser_parse_key(parser, event)
	case ini_PARSE_SECTION_VALUE_STATE:
		return ini_parser_parse_value(parser, event)
	default:
		panic("invalid parser state")
	}
	return false
}

func ini_parser_parse_document_start(parser *ini_parser_t, event *ini_event_t) bool {
	token := peek_token(parser)
	if token != nil {
		if token.typ == ini_DOCUMENT_START_TOKEN {
			skip_token(parser)
			parser.states = append(parser.states, ini_PARSE_DOCUMENT_END_STATE)
			parser.state = ini_PARSE_SECTION_FIRST_START_STATE
			*event = ini_event_t{
				typ:        ini_DOCUMENT_START_EVENT,
				start_mark: token.start_mark,
				end_mark:   token.end_mark,
			}
			return true
		} else {
			return ini_parser_set_parser_error(parser, "did not find expected <document-start>", token.start_mark)
		}
	} else {
		return ini_parser_set_parser_error(parser, "did not find expected <document-start>", parser.mark)
	}
}

// Parse the section:
func ini_parser_parse_section_start(parser *ini_parser_t, event *ini_event_t, first bool) bool {
	//defer trace("ini_parser_parse_section_start")
	token := peek_token(parser)
	if token != nil {
		if token.typ == ini_DOCUMENT_END_TOKEN {
			parser.state = parser.states[len(parser.states)-1]
			parser.states = parser.states[:len(parser.states)-1]
			*event = ini_event_t{
				typ:        ini_DOCUMENT_END_EVENT,
				start_mark: token.start_mark,
				end_mark:   token.end_mark,
			}
		} else {
			if first && token.typ == ini_KEY_TOKEN {
				parser.state = ini_PARSE_SECTION_ENTRY_STATE
				*event = ini_event_t{
					typ:        ini_SCALAR_EVENT,
					start_mark: token.start_mark,
					end_mark:   token.start_mark,
					value:      []byte(DEFAULT_SECTION),
					tag:        []byte(ini_STR_TAG),
				}
			} else if token.typ == ini_SECTION_START_TOKEN {
				skip_token(parser)
				token := peek_token(parser)
				if token != nil {
					if token.typ == ini_SCALAR_TOKEN {
						skip_token(parser)
						parser.state = ini_PARSE_SECTION_INHERIT_STATE
						*event = ini_event_t{
							typ:        ini_SCALAR_EVENT,
							start_mark: token.start_mark,
							end_mark:   token.end_mark,
							value:      []byte(token.value),
							tag:        []byte(ini_STR_TAG),
						}
					} else {
						return ini_parser_set_parser_error(parser, "did not find expected <scalar>", token.start_mark)
					}
				} else {
					return ini_parser_set_parser_error(parser, "did not find expected <scalar>", parser.mark)
				}
			} else {
				return ini_parser_set_parser_error(parser, "did not find expected <section-start> or <key>", token.start_mark)
			}
		}
		return true
	} else {
		return ini_parser_set_parser_error(parser, "did not find expected <document-end> or <section-start> or <key>", parser.mark)
	}
}

// Parse the section:
func ini_parser_parse_section_inherit(parser *ini_parser_t, event *ini_event_t) bool {
	//defer trace("ini_parser_parse_section_inherit")
	// SECTION-INHERIT Token (:)
	section_key := []byte(DEFAULT_SECTION)
	start_mark := parser.mark
	end_mark := parser.mark
	token := peek_token(parser)
	if token != nil {
		if token.typ == ini_SECTION_INHERIT_TOKEN {
			skip_token(parser)
			token = peek_token(parser)
			if token != nil {
				if token.typ == ini_SCALAR_TOKEN {
					start_mark = token.start_mark
					end_mark = token.end_mark
					section_key = token.value
					skip_token(parser)
				} else {
					return ini_parser_set_parser_error(parser, "did not find expected <scalar>", token.start_mark)
				}
			} else {
				return ini_parser_set_parser_error(parser, "did not find expected <scalar>", parser.mark)
			}
		}
		parser.state = ini_PARSE_SECTION_ENTRY_STATE
		*event = ini_event_t{
			typ:        ini_SECTION_INHERIT_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			value:      section_key,
			tag:        []byte(ini_STR_TAG),
		}
		return true
	} else {
		return false
	}
}
func ini_parser_parse_section_entry(parser *ini_parser_t, event *ini_event_t) bool {
	token := peek_token(parser)
	parser.state = ini_PARSE_SECTION_KEY_STATE
	if token != nil && token.typ == ini_SECTION_ENTRY_TOKEN {
		skip_token(parser)
		*event = ini_event_t{
			typ:        ini_SECTION_ENTRY_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
			tag:        []byte(ini_SECTION_TAG),
		}
	} else {
		*event = ini_event_t{
			typ:        ini_SECTION_ENTRY_EVENT,
			start_mark: parser.mark,
			end_mark:   parser.mark,
			tag:        []byte(ini_SECTION_TAG),
		}
	}

	return true
}

// Parse the productions:
// properties           ::= (KEY1.KEY2 = VALUE)
//
func ini_parser_parse_key(parser *ini_parser_t, event *ini_event_t) bool {
	token := peek_token(parser)
	if token != nil {
		if token.typ == ini_KEY_TOKEN {
			skip_token(parser)
			token := peek_token(parser)
			if token != nil {
                if token.typ == ini_SCALAR_TOKEN {
                    skip_token(parser)
                    parser.state = ini_PARSE_SECTION_VALUE_STATE
                    *event = ini_event_t{
                        typ:        ini_SCALAR_EVENT,
                        start_mark: token.start_mark,
                        end_mark:   token.end_mark,
                        value:      token.value,
                        style:      ini_style_t(token.style),
                    }
                } else {
                    return ini_parser_set_parser_error(parser, "did not find expected <scalar>", token.start_mark)
                }
            } else {
                return ini_parser_set_parser_error(parser, "did not find expected <scalar>", parser.mark)
            }
		} else {
			if token.typ != ini_SECTION_START_TOKEN && token.typ != ini_DOCUMENT_END_TOKEN {
				return ini_parser_set_parser_error(parser, "did not find expected <key> or <section-start>", token.start_mark)
			} else {
				parser.state = ini_PARSE_SECTION_START_STATE
				*event = ini_event_t{
					typ:        ini_SECTION_ENTRY_EVENT,
					start_mark: token.start_mark,
					end_mark:   token.end_mark,
				}
			}
		}
	} else {
		return ini_parser_set_parser_error(parser, "did not find expected <key> or <section-start>", parser.mark)
	}
	return true
}

func ini_parser_parse_value(parser *ini_parser_t, event *ini_event_t) bool {
	token := peek_token(parser)
	if token != nil {
		if token.typ == ini_MAP_TOKEN {
			skip_token(parser)
			parser.state = ini_PARSE_SECTION_KEY_STATE
			*event = ini_event_t{
				typ:        ini_MAPPING_EVENT,
				start_mark: token.start_mark,
				end_mark:   token.end_mark,
			}
		} else if token.typ == ini_VALUE_TOKEN {
			skip_token(parser)
			token := peek_token(parser)
			if token != nil && token.typ == ini_SCALAR_TOKEN {
				skip_token(parser)
				parser.state = ini_PARSE_SECTION_KEY_STATE
				*event = ini_event_t{
					typ:        ini_SCALAR_EVENT,
					start_mark: token.start_mark,
					end_mark:   token.end_mark,
					value:      token.value,
					style:      ini_style_t(token.style),
				}
			} else {
				return ini_parser_set_parser_error(parser, "did not find expected <scalar>", token.start_mark)
			}
		} else {
			return ini_parser_set_parser_error(parser, "did not find expected <value> or <map>", token.start_mark)
		}
	} else {
		return ini_parser_set_parser_error(parser, "did not find expected <value> or <map>", parser.mark)
	}
	return true
}

func ini_parser_process_empty_scalar(parser *ini_parser_t, event *ini_event_t, mark ini_mark_t) bool {
	*event = ini_event_t{
		typ:        ini_SCALAR_EVENT,
		start_mark: mark,
		end_mark:   mark,
		value:      nil, // Empty
		tag:        []byte(ini_NULL_TAG),
		style:      ini_style_t(ini_PLAIN_SCALAR_STYLE),
	}
	return true
}
