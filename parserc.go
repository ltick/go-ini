package ini

// The parser implements the following grammar:
//
// stream	    ::= STREAM-START document* STREAM-END
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
	parser.stream_end_produced = parser.tokens[parser.tokens_head].typ == ini_STREAM_END_TOKEN
	parser.tokens_head++
}

// Get the next event.
func ini_parser_parse(parser *ini_parser_t, event *ini_event_t) bool {
	// Erase the event object.
	*event = ini_event_t{}

	// No events after the end of the stream or error.
	if parser.stream_end_produced || parser.error != ini_NO_ERROR || parser.state == ini_PARSE_STREAM_END_STATE {
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
	trace("ini_parser_state_machine", "state:", parser.state.String())
	switch parser.state {
	case ini_PARSE_STREAM_START_STATE:
		return ini_parser_stream_start(parser, event)
	case ini_PARSE_DOCUMENT_START_STATE:
		return ini_parser_parse_document_start(parser, event)
	case ini_PARSE_DOCUMENT_CONTENT_STATE:
		return ini_parser_parse_document_content(parser, event)
	case ini_PARSE_DOCUMENT_END_STATE:
		return ini_parser_parse_document_end(parser, event)
	case ini_PARSE_SECTION_FIRST_ENTRY_STATE:
		return ini_parser_parse_section_entry(parser, event, true)
	case ini_PARSE_SECTION_ENTRY_STATE:
		return ini_parser_parse_section_entry(parser, event, false)
	case ini_PARSE_KEY_STATE:
		return ini_parser_parse_node(parser, event, true, false)
	default:
		panic("invalid parser state")
	}
	return false
}

func ini_parser_stream_start(parser *ini_parser_t, event *ini_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ != ini_STREAM_START_TOKEN {
		return ini_parser_set_parser_error(parser, "did not find expected <stream-start>", token.start_mark)
	}
	parser.state = ini_PARSE_DOCUMENT_START_STATE
	*event = ini_event_t{
		typ:        ini_STREAM_START_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
	}
	skip_token(parser)
	return true
}

// Parse the document:
func ini_parser_parse_document_start(parser *ini_parser_t, event *ini_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ != ini_STREAM_END_TOKEN {
		start_mark := token.start_mark
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != ini_DOCUMENT_START_TOKEN {
			return ini_parser_set_parser_error(parser, "did not find expected <document-start>", token.start_mark)
		}
		parser.states = append(parser.states, ini_PARSE_DOCUMENT_END_STATE)
		parser.state = ini_PARSE_DOCUMENT_CONTENT_STATE
		end_mark := token.end_mark

		*event = ini_event_t{
			typ:        ini_DOCUMENT_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
		}
		skip_token(parser)
	} else {
		// Parse the stream end.
		parser.state = ini_PARSE_STREAM_END_STATE
		*event = ini_event_t{
			typ:        ini_STREAM_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
		skip_token(parser)
	}
	return true
}

func ini_parser_parse_document_content(parser *ini_parser_t, event *ini_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ == ini_DOCUMENT_START_TOKEN ||
		token.typ == ini_DOCUMENT_END_TOKEN ||
		token.typ == ini_STREAM_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		return ini_parser_process_empty_scalar(parser, event,
			token.start_mark)
	}
	return ini_parser_parse_node(parser, event, true, false)
}

// Parse the document:
//
func ini_parser_parse_document_end(parser *ini_parser_t, event *ini_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	start_mark := token.start_mark
	end_mark := token.start_mark

	if token.typ == ini_DOCUMENT_END_TOKEN {
		end_mark = token.end_mark
		skip_token(parser)
	}

	parser.state = ini_PARSE_DOCUMENT_END_STATE
	*event = ini_event_t{
		typ:        ini_DOCUMENT_END_EVENT,
		start_mark: start_mark,
		end_mark:   end_mark,
	}
	return true
}

// Parse the section:
func ini_parser_parse_section_entry(parser *ini_parser_t, event *ini_event_t, first bool) bool {

	token := peek_token(parser)
	parser.marks = append(parser.marks, token.start_mark)
	if token == nil {
		return false
	}
	start_mark := token.start_mark
	end_mark := token.end_mark

	if token.typ == ini_SECTION_START_TOKEN {
		if first {
			parser.states = append(parser.states, ini_PARSE_SECTION_FIRST_ENTRY_STATE)
		} else {
			parser.states = append(parser.states, ini_PARSE_SECTION_ENTRY_STATE)
		}
		*event = ini_event_t{
			typ:        ini_SECTION_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			value:      token.value,
			implicit:   false,
			style:      ini_style_t(token.style),
		}
		skip_token(parser)
	} else if token.typ == ini_SECTION_END_TOKEN {
		*event = ini_event_t{
			typ:        ini_SECTION_END_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			value:      token.value,
			implicit:   false,
			style:      ini_style_t(token.style),
		}
		skip_token(parser)
	} else if token.typ == ini_KEY_TOKEN {
		if first {
			parser.states = append(parser.states, ini_PARSE_SECTION_FIRST_ENTRY_STATE)
		} else {
			parser.states = append(parser.states, ini_PARSE_SECTION_ENTRY_STATE)
		}
		// implicit
		*event = ini_event_t{
			typ:        ini_SECTION_START_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.start_mark,
			value:      token.value,
			implicit:   false,
			style:      ini_style_t(token.style),
		}

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

// Parse the productions:
// properties           ::= (KEY = VALUE)
//
func ini_parser_parse_node(parser *ini_parser_t, event *ini_event_t, block, indentless_sequence bool) bool {
    //defer trace("ini_parser_parse_node", "block:", block, "indentless_sequence:", indentless_sequence)()

	token := peek_token(parser)
	if token == nil {
		return false
	}

	start_mark := token.start_mark
	end_mark := token.start_mark
	if token.typ == ini_KEY_TOKEN {

	} else if token.typ == ini_VALUE_TOKEN {
		end_mark = token.end_mark

		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]

		*event = ini_event_t{
			typ:        ini_KEY_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			value:      token.value,
			implicit:   true,
			style:      ini_style_t(token.style),
		}
		skip_token(parser)
		return true
	}

	context := "while parsing a node"
	ini_parser_set_parser_error_context(parser, context, start_mark,
		"did not find expected node content", token.start_mark)
	return false
}

// Generate an empty scalar event.
func ini_parser_process_empty_scalar(parser *ini_parser_t, event *ini_event_t, mark ini_mark_t) bool {
    *event = ini_event_t{
        typ:        ini_SCALAR_EVENT,
        start_mark: mark,
        end_mark:   mark,
        value:      nil, // Empty
        implicit:   true,
        style:      ini_style_t(ini_PLAIN_SCALAR_STYLE),
    }
    return true
}
