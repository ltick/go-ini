package ini

import (
	"fmt"
)

// Introduction
// ************
//
// The process of transforming a INI stream into a sequence of events is
// divided on two steps: Scanning and Parsing.
//
// The Scanner transforms the input stream into a sequence of tokens, while the
// parser transform the sequence of tokens produced by the Scanner into a
// sequence of parsing events.
//
// The Scanner is rather clever and complicated. The Parser, on the contrary,
// is a straightforward implementation of a recursive-descendant parser (or,
// LL(1) parser, as it is usually called).
//
// Actually there are two issues of Scanning that might be called "clever", the
// rest is quite straightforward.  The issues are "block collection start" and
// "simple keys".  Both issues are explained below in details.
//
// Here the Scanning step is explained and implemented.  We start with the list
// of all the tokens produced by the Scanner together with short descriptions.
//
// Now, tokens:
//
//      STREAM-START                    # The stream start.
//      STREAM-END                      # The stream end.
//      DOCUMENT-START                  # The document start.
//      DOCUMENT-END                    # The document end.
//      SECTION-START             		# '[*]'
//      SECTION-END               		# ''
//      SECTION-INHERIT     			# ':'
//      KEY                             # nothing.
//      VALUE                           # '='
// 		SCALAR(value,style)             # A scalar.
//      COMMENT             			# '#', ';'
//
// The following two tokens are "virtual" tokens denoting the beginning and the
// end of the document:
//
//      DOCUMENT-START
//      DOCUMENT-END
//
// INI allowed spaces before and after KEY and VALUE and VALUE can wraps with '"' or '\''
// The following examples illustrate this case:
//
// 		1. Key and Value
//
//			key =   value
//
//		Tokens:
//
//      	KEY
//			SCALAR('key', plain)
//			VALUE
//			SCALAR('value', plain)
//
//		2. Value with wrapper
//
//			'key' = "value"
//
//		Tokens:
//
//      	KEY
//			SCALAR('key', single-quoted)
//			VALUE
//			SCALAR('value', double-quoted)
//
// A document can have many sections, The section start and end indicators are
// represented by:
//
//      SECTION-START
//      SECTION-END
//
// One section can inherit from another, The section inherit indicators are
// represented by:
//
//      SECTION-INHERIT
//
// In the following examples, we present whole documents together with the
// produced tokens.
//
//      1. A section:
//
//          [section]
//			key = value
//
//      Tokens:
//
//          DOCUMENT-START
//          SECTION-START
//			SCALAR('section', plain)
//			SECTION-END
//			KEY
//			SCALAR('key', plain)
//			VALUE
//			SCALAR('value', plain)
//          DOCUMENT-END
//
//      2. Several sections in a document:
//
//          [section]
//			key1 = "value1"
//          [another_section]
//			'key2' = 'value2'
//
//      Tokens:
//
//          DOCUMENT-START
//          SECTION-START
//			SCALAR('section', plain)
//			SECTION-END
//			KEY
//			SCALAR('key1', plain)
//			VALUE
//			SCALAR('value1', double-quoted)
//          SECTION-START
//			SCALAR('another_section', plain)
//			SECTION-END
//			KEY
//			SCALAR('key2', single-quoted)
//			VALUE
//			SCALAR('value2', single-quoted)
//          DOCUMENT-END
//
//      3. An sections inherit another :
//
//          [section]
//			key1 = 'value1'
//          [another_section:section]
//			"key1" = 'value2'
//
//      Tokens:
//
//          DOCUMENT-START
//          SECTION-START
//			SCALAR('section', plain)
//			SECTION-END
//			KEY
//			SCALAR('key1', plain)
//			VALUE
//			SCALAR('value1', double-quoted)
//          SECTION-START
//			SCALAR('another_section', plain)
//			SECTION-INHERIT
//			SCALAR('section', plain)
//			SECTION-END
//			KEY
//			SCALAR('key1', double-quoted)
//			VALUE
//			SCALAR('value2', single-quoted)
//          DOCUMENT-END
//
//
// also we can add comments in document.If the line cini_parser_parseontains
// '#' or ';' indicators, the following characters comment conent.
// The following examples illustrate this case:
//
//      1. Comment Line:
//
//			# comment_1
//
//      Tokens:
//
//			DOCUMENT-START
//          COMMENT
//			SCALAR('comment_1', plain)
//			DOCUMENT-END
//
//      2. Comment after section:
//
//          [section] ;section_comment
//
//      Tokens:
//
//          DOCUMENT-START
//          SECTION-START
//			SCALAR('section', plain)
//			SECTION-END
//			COMMENT
//			SCALAR('section_comment', plain)
//			DOCUMENT-END
//
// 		3. Comment after value:
//
//         key = value  #section_comment
//
//      Tokens:
//
//          DOCUMENT-START
//          KEY
//			SCALAR('key', plain)
//			VALUE
//			SCALAR('value', plain)
//			COMMENT
//			SCALAR('comment_1', plain)
//			DOCUMENT-END

// Ensure that the buffer contains the required number of characters.
// Return true on success, false on failure (reader error or memory error).
func cache(parser *ini_parser_t, length int) bool {
	// [Go] This was inlined: !cache(A, B) -> unread < B && !update(A, B)
	return parser.unread >= length || ini_parser_update_buffer(parser, length)
}

// Advance the buffer pointer.
func skip(parser *ini_parser_t) {
	parser.mark.index++
	parser.unread--
	parser.buffer_pos += width(parser.buffer[parser.buffer_pos])
}

func skip_line(parser *ini_parser_t) {
	if is_crlf(parser.buffer, parser.buffer_pos) {
		parser.mark.index += 2
		parser.mark.line++
		parser.unread -= 2
		parser.buffer_pos += 2
	} else if is_break(parser.buffer, parser.buffer_pos) {
		parser.mark.index++
		parser.mark.line++
		parser.unread--
		parser.buffer_pos += width(parser.buffer[parser.buffer_pos])
	}
}

// Copy a character to a string buffer and advance pointers.
func read(parser *ini_parser_t, s []byte) []byte {
	w := width(parser.buffer[parser.buffer_pos])
	if w == 0 {
		panic("invalid character sequence")
	}
	if len(s) == 0 {
		s = make([]byte, 0, 32)
	}
	if w == 1 && len(s)+w <= cap(s) {
		s = s[:len(s)+1]
		s[len(s)-1] = parser.buffer[parser.buffer_pos]
		parser.buffer_pos++
	} else {
		s = append(s, parser.buffer[parser.buffer_pos:parser.buffer_pos+w]...)
		parser.buffer_pos += w
	}
	parser.mark.index++
	parser.unread--
	return s
}

// Copy a line break character to a string buffer and advance pointers.
func read_line(parser *ini_parser_t, s []byte) []byte {
	buf := parser.buffer
	pos := parser.buffer_pos
	switch {
	case buf[pos] == '\r' && buf[pos+1] == '\n':
		// CR LF . LF
		s = append(s, '\n')
		parser.buffer_pos += 2
		parser.mark.index++
		parser.unread--
	case buf[pos] == '\r' || buf[pos] == '\n':
		// CR|LF . LF
		s = append(s, '\n')
		parser.buffer_pos += 1
	case buf[pos] == '\xC2' && buf[pos+1] == '\x85':
		// NEL . LF
		s = append(s, '\n')
		parser.buffer_pos += 2
	case buf[pos] == '\xE2' && buf[pos+1] == '\x80' && (buf[pos+2] == '\xA8' || buf[pos+2] == '\xA9'):
		// LS|PS . LS|PS
		s = append(s, buf[parser.buffer_pos:pos+3]...)
		parser.buffer_pos += 3
	default:
		return s
	}
	parser.mark.index++
	parser.mark.line++
	parser.unread--
	return s
}

// Set the scanner error and return false.
func ini_parser_set_scanner_error(parser *ini_parser_t, context string, context_mark ini_mark_t, problem string) bool {
	parser.error = ini_SCANNER_ERROR
	parser.context = context
	parser.context_mark = context_mark
	parser.problem = problem
	parser.problem_mark = parser.mark
	return false
}

func trace(args ...interface{}) func() {
	pargs := append([]interface{}{"+++"}, args...)
	fmt.Println(pargs...)
	pargs = append([]interface{}{"---"}, args...)
	return func() { fmt.Println(pargs...) }
}

// Initialize the scanner and produce the STREAM-START token.
func ini_parser_fetch_stream_start(parser *ini_parser_t) bool {

	// We have started.
	parser.stream_start_produced = true

	// Create the STREAM-START token and append it to the queue.
	token := ini_token_t{
		typ:        ini_STREAM_START_TOKEN,
		start_mark: parser.mark,
		end_mark:   parser.mark,
	}
	ini_insert_token(parser, -1, &token)
	return true
}

// Produce the STREAM-END token and shut down the scanner.
func ini_parser_fetch_stream_end(parser *ini_parser_t) bool {
    // Force new line.
    if parser.mark.column != 0 {
        parser.mark.column = 0
        parser.mark.line++
    }
	// Create the STREAM-END token and append it to the queue.
	token := ini_token_t{
		typ:        ini_STREAM_END_TOKEN,
		start_mark: parser.mark,
		end_mark:   parser.mark,
	}
	ini_insert_token(parser, -1, &token)
	return true
}

// Ensure that the tokens queue contains at least one token which can be
// returned to the Parser.
func ini_parser_fetch_more_tokens(parser *ini_parser_t) bool {
	// While we need more tokens to fetch, do it.
	for {
        // Check if we really need to fetch more tokens.
        need_more_tokens := false

        if parser.tokens_head == len(parser.tokens) {
            // Queue is empty.
            need_more_tokens = true
        }
        // We are finished.
        if !need_more_tokens {
            break
        }
		// Fetch the next token.
		if !ini_parser_fetch_next_token(parser) {
			return false
		}
	}

	parser.token_available = true
	return true
}

// The dispatcher for token fetchers.
func ini_parser_fetch_next_token(parser *ini_parser_t) bool {
	// Ensure that the buffer is initialized.
	if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
        return false
	}
	// Check if we just started scanning.  Fetch STREAM-START then.
	if !parser.stream_start_produced {
		return ini_parser_fetch_stream_start(parser)
	}

	// Eat whitespaces and comments until we reach the next token.
	if !ini_parser_scan_to_next_token(parser) {
		return false
	}

	// Is it the end of the stream?
	if is_z(parser.buffer, parser.buffer_pos) {
		return ini_parser_fetch_stream_end(parser)
	}

	buf := parser.buffer
	pos := parser.buffer_pos
    // Is it the document start indicator?
    if parser.mark.line == 0 && parser.mark.column == 0 {
        return ini_parser_fetch_document_indicator(parser, ini_DOCUMENT_START_TOKEN)
    }

    // Is it the document end indicator?
    if parser.eof {
        return ini_parser_fetch_document_indicator(parser, ini_DOCUMENT_END_TOKEN)
    }

	// Is it the section start indicator?
	if parser.mark.column == 0 && buf[pos] == '[' {
		return ini_parser_fetch_section(parser, ini_SECTION_START_TOKEN)
	}

	// Is it the flow mapping end indicator?
	if parser.buffer[parser.buffer_pos] == ':' {
		return ini_parser_fetch_section(parser, ini_SECTION_INHERIT_TOKEN)
	}

	// Is it the section end indicator?
	if buf[parser.buffer_pos] == ']' {
		return ini_parser_fetch_section(parser, ini_SECTION_END_TOKEN)
	}

	// Is it the node indicator?
	if parser.buffer[parser.buffer_pos] == '=' {
		return ini_parser_fetch_node(parser)
	}

	// If we don't determine the token type so far, it is an error.
	return ini_parser_set_scanner_error(parser,
		"while scanning for the next token", parser.mark,
		"found character that cannot start any token")
}

// Increase the flow level and resize the simple key list if needed.
func ini_parser_increase_level(parser *ini_parser_t) bool {
	// Increase the flow level.
	parser.level++
	return true
}

// Decrease the flow level.
func ini_parser_decrease_level(parser *ini_parser_t) bool {
	if parser.level > 0 {
		parser.level--
	}
	return true
}

// Produce the SECTION-START token.
func ini_parser_fetch_section(parser *ini_parser_t, typ ini_token_type_t) bool {
	parser.level = 0

	// Consume the token.
	start_mark := parser.mark
	skip(parser)
	end_mark := parser.mark

	switch typ {
	case ini_SECTION_START_TOKEN:
		// Create the FLOW-SECTION-START of FLOW-MAPPING-START token.
		token := ini_token_t{
			typ:        typ,
			start_mark: start_mark,
			end_mark:   end_mark,
		}
		// Append the token to the queue.
		ini_insert_token(parser, -1, &token)
	case ini_SECTION_INHERIT_TOKEN:
	case ini_SECTION_END_TOKEN:
		// Create the FLOW-SECTION-START of FLOW-MAPPING-START token.
		token := ini_token_t{
			typ:        typ,
			start_mark: start_mark,
			end_mark:   end_mark,
		}
		ini_insert_token(parser, -1, &token)
	}

	return true
}

// Produce the DOCUMENT-START or DOCUMENT-END token.
func ini_parser_fetch_document_indicator(parser *ini_parser_t, typ ini_token_type_t) bool {
	// Create the DOCUMENT-START or DOCUMENT-END token.
	token := ini_token_t{
		typ:        typ,
		start_mark: parser.mark,
		end_mark:   parser.mark,
	}
	// Append the token to the queue.
	ini_insert_token(parser, -1, &token)

	return true
}

// Produce the NODE token.
func ini_parser_fetch_node(parser *ini_parser_t) bool {

	// Consume the token.
	start_mark := parser.mark
	skip(parser)
	end_mark := parser.mark

	// Create the KEY token and append it to the queue.
	token := ini_token_t{
		typ:        ini_KEY_TOKEN,
		start_mark: start_mark,
		end_mark:   end_mark,
	}
	ini_insert_token(parser, -1, &token)

	token = ini_token_t{}
	ini_parser_scan_node_value(parser, &token)

	return true
}

// Scan a node value.
func ini_parser_scan_node_value(parser *ini_parser_t, token *ini_token_t) bool {
	// Eat the left quote.
	start_mark := parser.mark
	skip(parser)

	// Consume the content of the quoted scalar.
	var s, leading_break, trailing_breaks, whitespaces []byte
	var single bool
	for {
		if parser.unread < 4 && !ini_parser_update_buffer(parser, 4) {
			return false
		}

		// Check for EOF.
		if is_z(parser.buffer, parser.buffer_pos) {
			ini_parser_set_scanner_error(parser, "while scanning a quoted scalar",
				start_mark, "found unexpected end of stream")
			return false
		}

		// Consume non-blank characters.
		leading_blanks := false
		for !is_blankz(parser.buffer, parser.buffer_pos) {
			if parser.buffer[parser.buffer_pos] == '\'' && parser.buffer[parser.buffer_pos+1] == '\'' {
				// Is is an escaped single quote.
				s = append(s, '\'')
				skip(parser)
				skip(parser)
				single = true
			} else if parser.buffer[parser.buffer_pos] == '\'' {
				// It is a right single quote.
				single = true
				break
			} else if parser.buffer[parser.buffer_pos] == '"' {
				// It is a right double quote.
				break

			} else if parser.buffer[parser.buffer_pos] == '\\' && is_break(parser.buffer, parser.buffer_pos+1) {
				// It is an escaped line break.
				if parser.unread < 3 && !ini_parser_update_buffer(parser, 3) {
					return false
				}
				skip(parser)
				skip_line(parser)
				leading_blanks = true
				break

			} else if parser.buffer[parser.buffer_pos] == '\\' {
				// It is an escape sequence.
				code_length := 0

				// Check the escape character.
				switch parser.buffer[parser.buffer_pos+1] {
				case '0':
					s = append(s, 0)
				case 'a':
					s = append(s, '\x07')
				case 'b':
					s = append(s, '\x08')
				case 't', '\t':
					s = append(s, '\x09')
				case 'n':
					s = append(s, '\x0A')
				case 'v':
					s = append(s, '\x0B')
				case 'f':
					s = append(s, '\x0C')
				case 'r':
					s = append(s, '\x0D')
				case 'e':
					s = append(s, '\x1B')
				case ' ':
					s = append(s, '\x20')
				case '"':
					s = append(s, '"')
				case '\'':
					s = append(s, '\'')
				case '\\':
					s = append(s, '\\')
				case 'N': // NEL (#x85)
					s = append(s, '\xC2')
					s = append(s, '\x85')
				case '_': // #xA0
					s = append(s, '\xC2')
					s = append(s, '\xA0')
				case 'L': // LS (#x2028)
					s = append(s, '\xE2')
					s = append(s, '\x80')
					s = append(s, '\xA8')
				case 'P': // PS (#x2029)
					s = append(s, '\xE2')
					s = append(s, '\x80')
					s = append(s, '\xA9')
				case 'x':
					code_length = 2
				case 'u':
					code_length = 4
				case 'U':
					code_length = 8
				default:
					ini_parser_set_scanner_error(parser, "while parsing a quoted scalar",
						start_mark, "found unknown escape character")
					return false
				}

				skip(parser)
				skip(parser)

				// Consume an arbitrary escape code.
				if code_length > 0 {
					var value int

					// Scan the character value.
					if parser.unread < code_length && !ini_parser_update_buffer(parser, code_length) {
						return false
					}
					for k := 0; k < code_length; k++ {
						if !is_hex(parser.buffer, parser.buffer_pos+k) {
							ini_parser_set_scanner_error(parser, "while parsing a quoted scalar",
								start_mark, "did not find expected hexdecimal number")
							return false
						}
						value = (value << 4) + as_hex(parser.buffer, parser.buffer_pos+k)
					}

					// Check the value and write the character.
					if (value >= 0xD800 && value <= 0xDFFF) || value > 0x10FFFF {
						ini_parser_set_scanner_error(parser, "while parsing a quoted scalar",
							start_mark, "found invalid Unicode character escape code")
						return false
					}
					if value <= 0x7F {
						s = append(s, byte(value))
					} else if value <= 0x7FF {
						s = append(s, byte(0xC0+(value>>6)))
						s = append(s, byte(0x80+(value&0x3F)))
					} else if value <= 0xFFFF {
						s = append(s, byte(0xE0+(value>>12)))
						s = append(s, byte(0x80+((value>>6)&0x3F)))
						s = append(s, byte(0x80+(value&0x3F)))
					} else {
						s = append(s, byte(0xF0+(value>>18)))
						s = append(s, byte(0x80+((value>>12)&0x3F)))
						s = append(s, byte(0x80+((value>>6)&0x3F)))
						s = append(s, byte(0x80+(value&0x3F)))
					}

					// Advance the pointer.
					for k := 0; k < code_length; k++ {
						skip(parser)
					}
				}
			} else {
				// It is a non-escaped non-blank character.
				s = read(parser, s)
			}
			if parser.unread < 2 && !ini_parser_update_buffer(parser, 2) {
				return false
			}
		}

		// Check if we are at the end of the scalar.
		if single {
			if parser.buffer[parser.buffer_pos] == '\'' {
				break
			}
		} else {
			if parser.buffer[parser.buffer_pos] == '"' {
				break
			}
		}

		// Consume blank characters.
		if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
			return false
		}

		for is_blank(parser.buffer, parser.buffer_pos) || is_break(parser.buffer, parser.buffer_pos) {
			if is_blank(parser.buffer, parser.buffer_pos) {
				// Consume a space or a tab character.
				if !leading_blanks {
					whitespaces = read(parser, whitespaces)
				} else {
					skip(parser)
				}
			} else {
				if parser.unread < 2 && !ini_parser_update_buffer(parser, 2) {
					return false
				}

				// Check if it is a first line break.
				if !leading_blanks {
					whitespaces = whitespaces[:0]
					leading_break = read_line(parser, leading_break)
					leading_blanks = true
				} else {
					trailing_breaks = read_line(parser, trailing_breaks)
				}
			}
			if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
				return false
			}
		}

		// Join the whitespaces or fold line breaks.
		if leading_blanks {
			// Do we need to fold line breaks?
			if len(leading_break) > 0 && leading_break[0] == '\n' {
				if len(trailing_breaks) == 0 {
					s = append(s, ' ')
				} else {
					s = append(s, trailing_breaks...)
				}
			} else {
				s = append(s, leading_break...)
				s = append(s, trailing_breaks...)
			}
			trailing_breaks = trailing_breaks[:0]
			leading_break = leading_break[:0]
		} else {
			s = append(s, whitespaces...)
			whitespaces = whitespaces[:0]
		}
	}

	// Eat the right quote.
	skip(parser)
	end_mark := parser.mark

	// Create a token.
	*token = ini_token_t{
		typ:        ini_VALUE_TOKEN,
		start_mark: start_mark,
		end_mark:   end_mark,
		value:      s,
		style:      ini_SINGLE_QUOTED_SCALAR_STYLE,
	}
	if !single {
		token.style = ini_DOUBLE_QUOTED_SCALAR_STYLE
	}
	return true
}

// Eat whitespaces and comments until the next token is found.
func ini_parser_scan_to_next_token(parser *ini_parser_t) bool {
	// Until the next token is not found.
	for {
		// Allow the BOM mark to start a line.
		if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
			return false
		}
		if is_bom(parser.buffer, parser.buffer_pos) {
			skip(parser)
		}

		// Eat whitespaces.
		if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
			return false
		}
		for parser.buffer[parser.buffer_pos] == ' ' || parser.buffer[parser.buffer_pos] == '\t' {
			skip(parser)
			if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
				return false
			}
		}

		// Eat a comment until a line break.
		if parser.buffer[parser.buffer_pos] == '#' || parser.buffer[parser.buffer_pos] == ';' {
			for !is_breakz(parser.buffer, parser.buffer_pos) {
				skip(parser)
				if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
					return false
				}
			}
		}

		// If it is a line break, eat it.
		if is_break(parser.buffer, parser.buffer_pos) {
			if parser.unread < 2 && !ini_parser_update_buffer(parser, 2) {
				return false
			}
			skip_line(parser)
		} else {
			break // We have found a token.
		}
	}

	return true
}
