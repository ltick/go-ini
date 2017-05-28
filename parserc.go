package ini

import "fmt"

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
	case ini_PARSE_DOCUMENT_END_STATE:
		return ini_parser_parse_document_content(parser, event)
	case ini_PARSE_SECTION_FIRST_ENTRY_STATE:
		return ini_parser_parse_section_entry(parser, event, true)
	case ini_PARSE_SECTION_ENTRY_STATE:
		return ini_parser_parse_section_entry(parser, event, false)
	case ini_PARSE_ELEMENT_KEY_STATE:
		return ini_parser_parse_item_key(parser, event)
	case ini_PARSE_ELEMENT_VALUE_STATE:
		return ini_parser_parse_item_value(parser, event)
	default:
		panic("invalid parser state")
	}
	return false
}

func ini_parser_parse_document_start(parser *ini_parser_t, event *ini_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ != ini_DOCUMENT_START_TOKEN {
		return ini_parser_set_parser_error(parser, "did not find expected <stream-start>", token.start_mark)
	}
	parser.states = append(parser.states, ini_PARSE_DOCUMENT_END_STATE)
	parser.state = ini_PARSE_DOCUMENT_END_STATE
	*event = ini_event_t{
		typ:        ini_DOCUMENT_START_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
	}
	skip_token(parser)
	return true
}

// Parse the document:
func ini_parser_parse_document_content(parser *ini_parser_t, event *ini_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
    fmt.Println("ini_parser_parse_document_content")

	if token.typ == ini_DOCUMENT_START_TOKEN ||
		token.typ == ini_DOCUMENT_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		return ini_parser_process_empty_element(parser, event,
			token.start_mark)
	}
    fmt.Println(len(parser.states))
    fmt.Println(parser.states[0])

	if token.typ == ini_SECTION_START_TOKEN {
		start_mark := token.start_mark
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != ini_DOCUMENT_START_TOKEN {
			return ini_parser_set_parser_error(parser, "did not find expected <document>", token.start_mark)
		}
		parser.state = ini_PARSE_SECTION_FIRST_ENTRY_STATE
		end_mark := token.end_mark

		*event = ini_event_t{
			typ:        ini_SECTION_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
		}
		skip_token(parser)
	} else if token.typ != ini_DOCUMENT_END_TOKEN {
		parser.state = ini_PARSE_SECTION_FIRST_ENTRY_STATE

		*event = ini_event_t{
			typ:        ini_SECTION_START_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
			value:      []byte("default"),
		}
	} else {
		// Parse the document end.
		parser.state = ini_PARSE_DOCUMENT_END_STATE
		*event = ini_event_t{
			typ:        ini_DOCUMENT_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
		skip_token(parser)
	}
	return true
}

// Parse the section:
func ini_parser_parse_section_entry(parser *ini_parser_t, event *ini_event_t, first bool) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
    fmt.Println("=====")
    fmt.Println(string(token.value))
	start_mark := token.start_mark
	token = peek_token(parser)
	if token == nil {
		return false
	}
	if !first && token.typ != ini_SECTION_START_TOKEN {
		return ini_parser_set_parser_error(parser, "did not find expected <section> or <scalar>", token.start_mark)
	}
	end_mark := token.end_mark
	if token.typ == ini_SECTION_START_TOKEN {
        parser.states = append(parser.states, ini_PARSE_SECTION_ENTRY_STATE)
        token := peek_token(parser)
        if token == nil {
            return false
        }
        // SECTION-INHERIT Token (:)
        if token.typ == ini_SECTION_INHERIT_TOKEN {
            parser.states = append(parser.states, ini_PARSE_SECTION_INHERIT_STATE)
            token = peek_token(parser)
            if token == nil {
                return false
            }
        }
        if token.typ == ini_SECTION_END_TOKEN {
			parser.state = ini_PARSE_SECTION_START_STATE
			*event = ini_event_t{
				typ:        ini_SECTION_ENTRY_EVENT,
				start_mark: start_mark,
				end_mark:   end_mark,
			}
            return true
        }
        return false
	} else {
        parser.states = append(parser.states, ini_PARSE_SECTION_END_STATE)
		parser.state = ini_PARSE_ELEMENT_KEY_STATE
		*event = ini_event_t{
			typ:        ini_ELEMENT_KEY_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.start_mark,
			value:      token.value,
			style:      ini_style_t(token.style),
		}
	}
	return true
}

func ini_parser_parse_section(parser *ini_parser_t, event *ini_event_t) bool {


}

func ini_parser_parse_item_key(parser *ini_parser_t, event *ini_event_t) bool {
	//defer trace("ini_parser_parse_item_key")()

	token := peek_token(parser)
	if token == nil {
		return false
	}
    if token.typ != ini_DOCUMENT_END_TOKEN {
		parser.state = ini_PARSE_ELEMENT_KEY_STATE
		*event = ini_event_t{
			typ:        ini_ELEMENT_KEY_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.start_mark,
			value:      token.value,
			style:      ini_style_t(token.style),
		}
		skip_token(parser)
		return true
	} else  {
		parser.state = ini_PARSE_DOCUMENT_END_STATE
		*event = ini_event_t{
			typ:        ini_DOCUMENT_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
		skip_token(parser)
	}

	return false
}

// Parse the productions:
// properties           ::= (KEY = VALUE)
//
func ini_parser_parse_item_value(parser *ini_parser_t, event *ini_event_t) bool {
	//defer trace("ini_parser_parse_item_value")()
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ == ini_VALUE_TOKEN {
		parser.state = ini_PARSE_ELEMENT_VALUE_STATE
		*event = ini_event_t{
			typ:        ini_ELEMENT_VALUE_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
			value:      token.value,
			style:      ini_style_t(token.style),
		}
		skip_token(parser)
		return true
	} else if token.typ == ini_DOCUMENT_END_TOKEN {
		parser.state = ini_PARSE_DOCUMENT_END_STATE
		*event = ini_event_t{
			typ:        ini_DOCUMENT_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
		skip_token(parser)
	}

	return false
}

func ini_parser_process_empty_element(parser *ini_parser_t, event *ini_event_t, mark ini_mark_t) bool {
    *event = ini_event_t{
        typ:        ini_ELEMENT_VALUE_EVENT,
        start_mark: mark,
        end_mark:   mark,
        value:      nil, // Empty
        style:      ini_style_t(ini_PLAIN_SCALAR_STYLE),
    }
    return true
}
