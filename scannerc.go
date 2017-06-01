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
//      DOCUMENT-START                  # The stream start.
//      DOCUMENT-END                    # The stream end.
//      SECTION-START            		# '['
//      SECTION-INHERIT     			# ':'
//      SECTION-END               		# ']'
//     	KEY                           	# nothing.
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
//			key = value
//
//		Tokens:
//
//			KEY
// 			SCALAR('key', plain)
//			VALUE
//			SCALAR('value', plain)
//
//		2. Value with wrapper
//
//			key = "value"
//
//		Tokens:
//
//			KEY
//			SCALAR('key', plain)
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
//          SCALAR('section', plain)
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
//          DOCUMENT
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
//          DOCUMENT-END
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
//          DOCUMENT-END
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
//          DOCUMENT-END

// Ensure that the buffer contains the required number of characters.
// Return true on success, false on failure (reader error or memory error).
func cache(parser *ini_parser_t, length int) bool {
	// [Go] This was inlined: !cache(A, B) -> unread < B && !update(A, B)
	return parser.unread >= length || ini_parser_update_buffer(parser, length)
}

// Advance the buffer pointer.
func skip(parser *ini_parser_t) {
	parser.mark.index++
	parser.mark.column++
	parser.unread--
	parser.buffer_pos += width(parser.buffer[parser.buffer_pos])
}

func skip_line(parser *ini_parser_t) {
	if is_crlf(parser.buffer, parser.buffer_pos) {
		parser.mark.index += 2
		parser.mark.column = 0
		parser.mark.line++
		parser.unread -= 2
		parser.buffer_pos += 2
	} else if is_break(parser.buffer, parser.buffer_pos) {
		parser.mark.index++
		parser.mark.column = 0
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
	parser.mark.column++
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
	parser.mark.column = 0
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

// Initialize the scanner and produce the DOCUMENT-START token.
func ini_parser_fetch_document_start(parser *ini_parser_t) bool {

	// We have started.
	parser.document_start_produced = true

	// Create the DOCUMENT-START token and append it to the queue.
	token := ini_token_t{
		typ:        ini_DOCUMENT_START_TOKEN,
		start_mark: parser.mark,
		end_mark:   parser.mark,
	}
	ini_insert_token(parser, -1, &token)
	return true
}

// Produce the DOCUMENT-END token and shut down the scanner.
func ini_parser_fetch_document_end(parser *ini_parser_t) bool {
	// Force new line.
	if parser.mark.column != 0 {
		parser.mark.column = 0
		parser.mark.line++
	}
	// Create the DOCUMENT-END token and append it to the queue.
	token := ini_token_t{
		typ:        ini_DOCUMENT_END_TOKEN,
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
	// Check if we just started scanning.  Fetch DOCUMENT-START then.
	if !parser.document_start_produced {
		return ini_parser_fetch_document_start(parser)
	}

	// Eat whitespaces and comments until we reach the next token.
	if !ini_parser_scan_to_next_token(parser) {
		return false
	}
	// Is it the end of the document?
	if is_z(parser.buffer, parser.buffer_pos) {
		return ini_parser_fetch_document_end(parser)
	}

	// Is it the section start indicator?
	if parser.mark.column == 0 && parser.buffer[parser.buffer_pos] == '[' {
		return ini_parser_fetch_section(parser)
	}

	// Is it the item value indicator?
	if parser.buffer[parser.buffer_pos] == '=' {
        for is_blankz(parser.buffer, parser.buffer_pos) {
            parser.buffer_pos++
        }
		if parser.buffer[parser.buffer_pos] == '\'' {
			// Is it a single-quoted scalar?
			return ini_parser_fetch_element_value(parser, true)
		} else if parser.buffer[parser.buffer_pos] == '"' {
			// Is it a double-quoted scalar?
			return ini_parser_fetch_element_value(parser, false)
		} else {
			return ini_parser_fetch_plain_element_value(parser)
		}
	}

	for is_blankz(parser.buffer, parser.buffer_pos) {
		parser.buffer_pos++
	}
	if parser.buffer[parser.buffer_pos] == '\'' {
		// Is it a single-quoted scalar?
		return ini_parser_fetch_element_key(parser, true)
	} else if parser.buffer[parser.buffer_pos] == '"' {
		// Is it a double-quoted scalar?
		return ini_parser_fetch_element_key(parser, false)
	} else {
		return ini_parser_fetch_plain_element_key(parser)
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

// Produce the SECTION token.
func ini_parser_fetch_section(parser *ini_parser_t) bool {
	// Eat '['
	if parser.buffer[parser.buffer_pos] == '[' {
        var s []byte
        start_mark := parser.mark
        s = read(parser, s)
		if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
			return false
		}
		end_mark := parser.mark
		token := ini_token_t{
			typ:        ini_SECTION_START_TOKEN,
			start_mark: start_mark,
			end_mark:   end_mark,
			value:      s,
		}
		ini_insert_token(parser, -1, &token)
	}
	// try to scan name.
	for parser.buffer[parser.buffer_pos] != ']' && parser.buffer[parser.buffer_pos] != ':' &&
		parser.buffer[parser.buffer_pos] != '\n' {
		ini_parser_fetch_plain_element_value(parser)
	}
	// Check for ':' and eat it.
	if parser.buffer[parser.buffer_pos] == ':' {
        var s []byte
        start_mark := parser.mark
        s = read(parser, s)
        if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
			return false
		}
		end_mark := parser.mark
		token := ini_token_t{
			typ:        ini_SECTION_INHERIT_TOKEN,
			start_mark: start_mark,
			end_mark:   end_mark,
			value:      s,
		}
		ini_insert_token(parser, -1, &token)
	}

	// Check for ']' and eat it.
	if parser.buffer[parser.buffer_pos] == ']' {
        var s []byte
        start_mark := parser.mark
        s = read(parser, s)
        if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
			return false
		}
		end_mark := parser.mark
		token := ini_token_t{
			typ:        ini_SECTION_END_TOKEN,
			start_mark: start_mark,
			end_mark:   end_mark,
			value:      s,
		}
		ini_insert_token(parser, -1, &token)
	} else if is_crlf(parser.buffer, parser.buffer_pos) {
		token := ini_token_t{
			typ:        ini_SECTION_END_TOKEN,
			start_mark: parser.mark,
			end_mark:   parser.mark,
			value:      parser.buffer[parser.mark.index:parser.mark.index],
		}
		ini_insert_token(parser, -1, &token)
	}

	return true
}

// Produce the KEY token.
func ini_parser_fetch_plain_element_key(parser *ini_parser_t) bool {
	token := ini_token_t{
		typ:        ini_SECTION_KEY_TOKEN,
		start_mark: parser.mark,
		end_mark:   parser.mark,
		value:      parser.buffer[parser.mark.index:parser.mark.index],
	}
	ini_insert_token(parser, -1, &token)

    // key must start with alpha([0-9a-zA-Z_-])
    if !is_alpha(parser.buffer, parser.buffer_pos) {
        failf("found invaild character(%c) that cannot start with while scanning for key", parser.buffer[parser.buffer_pos])
    }

	if !ini_parser_fetch_plain_scalar(parser) {
		return false
	}

	return true
}

// Produce the NODE token.
func ini_parser_fetch_element_key(parser *ini_parser_t, single bool) bool {
	token := ini_token_t{
		typ:        ini_SECTION_KEY_TOKEN,
		start_mark: parser.mark,
		end_mark:   parser.mark,
		value:      parser.buffer[parser.mark.index:parser.mark.index],
	}
	ini_insert_token(parser, -1, &token)

    // key must start with alpha([0-9a-zA-Z_-])
    if !is_alpha(parser.buffer, parser.buffer_pos+1) {
        failf("found invaild character(%c) that cannot start with while scanning for key", parser.buffer[parser.buffer_pos])
    }

	if !ini_parser_fetch_scalar(parser, single) {
		return false
	}

	return true
}

// Produce the VALUE(...,plain) token.
func ini_parser_fetch_plain_element_value(parser *ini_parser_t) bool {
    // Eat '='
    if parser.buffer[parser.buffer_pos] == '=' {
        var s []byte
        start_mark := parser.mark
        s = read(parser, s)
        if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
            return false
        }
        end_mark := parser.mark
        token := ini_token_t{
            typ:        ini_SECTION_VALUE_TOKEN,
            start_mark: start_mark,
            end_mark:   end_mark,
            value:      s,
        }
        ini_insert_token(parser, -1, &token)
    }
	var token ini_token_t
	if !ini_parser_scan_plain_scalar(parser, &token) {
		return false
	}
	ini_insert_token(parser, -1, &token)
	return true
}

// Produce the VALUE token.
func ini_parser_fetch_element_value(parser *ini_parser_t, single bool) bool {
    // Eat '='
    if parser.buffer[parser.buffer_pos] == '=' {
        var s []byte
        start_mark := parser.mark
        s = read(parser, s)
        if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
            return false
        }
        end_mark := parser.mark
        token := ini_token_t{
            typ:        ini_SECTION_VALUE_TOKEN,
            start_mark: start_mark,
            end_mark:   end_mark,
            value:      s,
        }
        ini_insert_token(parser, -1, &token)
    }
	var token ini_token_t
	if !ini_parser_scan_scalar(parser, &token, single) {
		return false
	}
	ini_insert_token(parser, -1, &token)
	return true
}

// Produce the SCALAR(...,plain) token.
func ini_parser_fetch_plain_scalar(parser *ini_parser_t) bool {
	// Create the SCALAR token and append it to the queue.
	var token ini_token_t
	if !ini_parser_scan_plain_scalar(parser, &token) {
		return false
	}
	ini_insert_token(parser, -1, &token)
	return true
}

// Scan a plain scalar.
func ini_parser_scan_plain_scalar(parser *ini_parser_t, token *ini_token_t) bool {
	start_mark := parser.mark

	var s []byte
	// Consume the content of the plain scalar.
	for {
		// Check for a comment.
		if parser.buffer[parser.buffer_pos] == '#' {
			break
		}
		// Consume blank characters.
		for is_blank(parser.buffer, parser.buffer_pos) {
			skip(parser)
		}

		// Consume non-break characters.
		for !is_breakz(parser.buffer, parser.buffer_pos) {
			// Check for indicators that may end a plain scalar.
			if parser.buffer[parser.buffer_pos] == ':' || parser.buffer[parser.buffer_pos] == '=' ||
				parser.buffer[parser.buffer_pos] == '[' || parser.buffer[parser.buffer_pos] == ']' {
				break
			}

			// Copy the character.
			s = read(parser, s)

			if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
				return false
			}
		}

		// Consume blank characters.
		for is_blank(parser.buffer, parser.buffer_pos) {
			skip(parser)
		}

		// Is it the end?
		if !is_break(parser.buffer, parser.buffer_pos) {
			break
		}
	}
	end_mark := parser.mark

	// Create a token.
	*token = ini_token_t{
		typ:        ini_SCALAR_TOKEN,
		start_mark: start_mark,
		end_mark:   end_mark,
		value:      s,
		style:      ini_PLAIN_SCALAR_STYLE,
	}

	return true
}

// Produce the SCALAR(...,plain) token.
func ini_parser_fetch_scalar(parser *ini_parser_t, single bool) bool {
	// Create the SCALAR token and append it to the queue.
	var token ini_token_t
	if !ini_parser_scan_scalar(parser, &token, single) {
		return false
	}
	ini_insert_token(parser, -1, &token)
	return true
}

// Scan a node value.
func ini_parser_scan_scalar(parser *ini_parser_t, token *ini_token_t, single bool) bool {
	start_mark := parser.mark

	var s []byte
	// Consume the content of the quoted scalar.
	for {
		// Check for a comment.
		if parser.buffer[parser.buffer_pos] == '#' {
			break
		}
		if parser.buffer[parser.buffer_pos] == ':' || parser.buffer[parser.buffer_pos] == '=' ||
			parser.buffer[parser.buffer_pos] == '[' || parser.buffer[parser.buffer_pos] == ']' {
			break
		}
		for !is_breakz(parser.buffer, parser.buffer_pos) {
			if single && parser.buffer[parser.buffer_pos] == '\'' && parser.buffer[parser.buffer_pos+1] == '\'' {
				// Is is an escaped single quote.
				s = append(s, '\'')
				skip(parser)
				skip(parser)
			} else if single && parser.buffer[parser.buffer_pos] == '\'' {
				// It is a right single quote.
				break
			} else if !single && parser.buffer[parser.buffer_pos] == '"' {
				// It is a right double quote.
				break

			} else if !single && parser.buffer[parser.buffer_pos] == '\\' && is_break(parser.buffer, parser.buffer_pos+1) {
				// It is an escaped line break.
				if parser.unread < 3 && !ini_parser_update_buffer(parser, 3) {
					return false
				}
				skip(parser)
				skip_line(parser)
				break

			} else if !single && parser.buffer[parser.buffer_pos] == '\\' {
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

		// Is it the end?
		if is_breakz(parser.buffer, parser.buffer_pos) {
			break
		}

		// Consume blank characters.
		if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
			return false
		}
	}

	// Eat the right quote.
	skip(parser)
	end_mark := parser.mark

	// Create a token.
	*token = ini_token_t{
		typ:        ini_SECTION_VALUE_TOKEN,
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
		if parser.mark.column == 0 && is_bom(parser.buffer, parser.buffer_pos) {
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
