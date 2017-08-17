package ini

import (
	"bytes"
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
//      SECTION-ENTRY               	# ']'
//      KEY                             # nothing
// 		MAP 						    # '.'
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
// The first level key of '.' could ignored
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
// 		3.  Map Key and Value
//
//		    key1.key = value
//
//		Tokens:
//
//			KEY
// 			SCALAR('key1', plain)
//          MAP
//          KEY
// 			SCALAR('key', plain)
//			VALUE
//			SCALAR('value', plain)
//
//		4. Value with wrapper
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
//
// A document can have many sections, The section start and end indicators are
// represented by:
//
//      SECTION-START
//      SECTION-ENTRY
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
//			SECTION-ENTRY
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
//			SECTION-ENTRY
//			KEY
//			SCALAR('key1', plain)
//			VALUE
//			SCALAR('value1', double-quoted)
//          SECTION-START
//			SCALAR('another_section', plain)
//			SECTION-ENTRY
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
//			SECTION-ENTRY
//			KEY
//			SCALAR('key1', plain)
//			VALUE
//			SCALAR('value1', double-quoted)
//          SECTION-START
//			SCALAR('another_section', plain)
//			SECTION-INHERIT
//			SCALAR('section', plain)
//			SECTION-ENTRY
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
//			SECTION-ENTRY
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
		return ini_parser_fetch_section_start(parser)
	}
	if parser.buffer[parser.buffer_pos] == ':' {
		return ini_parser_fetch_section_inherit(parser)
	}
	if parser.buffer[parser.buffer_pos] == ']' {
		return ini_parser_fetch_section_entry(parser)
	}

	// Is it the item value indicator?
	if parser.buffer[parser.buffer_pos] == '=' {
		return ini_parser_fetch_value(parser)
	}

	return ini_parser_fetch_key(parser)
}

// Increase the flow level and resize the simple key list if needed.
func ini_parser_increase_key_level(parser *ini_parser_t) bool {
	// Increase the flow level.
	parser.key_level++
	return true
}

// Decrease the flow level.
func ini_parser_decrease_key_level(parser *ini_parser_t) bool {
	if parser.key_level > 0 {
		parser.key_level--
	}
	return true
}

// Produce the SECTION token.
func ini_parser_fetch_section_start(parser *ini_parser_t) bool {
	if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
		return false
	}

	// Consume the token.
	start_mark := parser.mark
	skip(parser)
	end_mark := parser.mark
	section_start_token := ini_token_t{
		typ:        ini_SECTION_START_TOKEN,
		start_mark: start_mark,
		end_mark:   end_mark,
		value:      []byte("["),
	}
	ini_insert_token(parser, -1, &section_start_token)
	// Produce the SCALAR(...,plain) token.
	// Create the SCALAR token and append it to the queue.
	var scalar_token ini_token_t
	if !ini_parser_fetch_section_key(parser, &scalar_token) {
		return false
	}
	ini_insert_token(parser, -1, &scalar_token)

	return true
}

func ini_parser_fetch_section_inherit(parser *ini_parser_t) bool {
	// Consume the token.
	start_mark := parser.mark
	skip(parser)
	end_mark := parser.mark
	section_inherit_token := ini_token_t{
		typ:        ini_SECTION_INHERIT_TOKEN,
		start_mark: start_mark,
		end_mark:   end_mark,
		value:      []byte(":"),
	}
	ini_insert_token(parser, -1, &section_inherit_token)
	// Produce the SCALAR(...,plain) token.
	// Create the SCALAR token and append it to the queue.
	var scalar_token ini_token_t
	if !ini_parser_fetch_section_key(parser, &scalar_token) {
		return false
	}
	ini_insert_token(parser, -1, &scalar_token)
	return true
}

func ini_parser_fetch_section_entry(parser *ini_parser_t) bool {
	if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
		return false
	}

	// Consume the token.
	start_mark := parser.mark
	skip(parser)
	end_mark := parser.mark
	if !is_break(parser.buffer, parser.buffer_pos) {
		return ini_parser_set_scanner_error(parser,
			"while scanning for the section entry", parser.mark,
			"must have a line break before the first section key")
	}
	token := ini_token_t{
		typ:        ini_SECTION_ENTRY_TOKEN,
		start_mark: start_mark,
		end_mark:   end_mark,
		value:      []byte("]"),
	}
	ini_insert_token(parser, -1, &token)
	return true
}

func ini_parser_fetch_section_key(parser *ini_parser_t, token *ini_token_t) bool {
    // Eat whitespaces.
    if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
        return false
    }
    for is_blank(parser.buffer, parser.buffer_pos) {
        skip(parser)
        if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
            return false
        }
    }
	start_mark := parser.mark
	var s []byte
	// Consume the content of the plain scalar.
	for {
		if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
			return false
		}
		if is_break(parser.buffer, parser.buffer_pos) {
			break
		}
		if parser.buffer[parser.buffer_pos] == ':' || parser.buffer[parser.buffer_pos] == '[' || parser.buffer[parser.buffer_pos] == ']' {
			break
		}
		if !is_alpha(parser.buffer, parser.buffer_pos) {
			return ini_parser_set_scanner_error(parser,
				"while scanning for the section key", parser.mark,
				"found character("+string([]byte{parser.buffer[parser.buffer_pos]})+") that cannot start for any section key")
		}
		// Copy the character.
		s = read(parser, s)
	}
	end_mark := parser.mark
	// Trim blank characters.
	s = bytes.Trim(s, " ")
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

func ini_parser_fetch_key(parser *ini_parser_t) bool {
    // Eat whitespaces.
    if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
        return false
    }
    for is_blank(parser.buffer, parser.buffer_pos) {
        skip(parser)
        if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
            return false
        }
    }
	// Produce the SCALAR(...,plain) token.
	var key_token ini_token_t
	if parser.buffer[parser.buffer_pos] == '\'' {
		// key must start with alpha([0-9a-zA-Z_-])
		if !is_alpha(parser.buffer, parser.buffer_pos+1) && parser.buffer[parser.buffer_pos+1] != '~' {
			return ini_parser_set_scanner_error(parser,
				"while scanning for the scalar", parser.mark,
				"found character("+string([]byte{parser.buffer[parser.buffer_pos+1]})+") that cannot start for any key")
		}
		if !ini_parser_scan_scalar(parser, &key_token, true) {
			return false
		}
	} else if parser.buffer[parser.buffer_pos] == '"' {
		if !ini_parser_scan_scalar(parser, &key_token, false) {
			return false
		}
	} else {
		if !ini_parser_scan_plain_scalar(parser, &key_token) {
			return false
		}
	}
	keys := bytes.Split(key_token.value, []byte("."))
	key_len := len(keys)
	key_start_mark := key_token.start_mark
	for i := 0; i < key_len; i++ {
		if len(keys[i]) == 0 {
			return ini_parser_set_scanner_error(parser,
				"while scanning for the key", parser.mark,
				"empty map key")
		}
		// key
		key_token := ini_token_t{
			typ:        ini_KEY_TOKEN,
			start_mark: key_start_mark,
			end_mark:   key_start_mark,
		}
		ini_insert_token(parser, -1, &key_token)
		key_end_mark := key_start_mark
		key_end_mark.index = key_start_mark.index + len(keys[i])
		scalar_token := ini_token_t{
			typ:        ini_SCALAR_TOKEN,
			start_mark: key_start_mark,
			end_mark:   key_end_mark,
			value:      keys[i],
			style:      ini_PLAIN_SCALAR_STYLE,
		}
		ini_insert_token(parser, -1, &scalar_token)
		if i < key_len-1 {
			// map
			key_start_mark = key_end_mark
			key_end_mark.index = key_start_mark.index + 1
			map_token := ini_token_t{
				typ:        ini_MAP_TOKEN,
				start_mark: key_start_mark,
				end_mark:   key_end_mark,
				value:      []byte("."),
				style:      ini_PLAIN_SCALAR_STYLE,
			}
			ini_insert_token(parser, -1, &map_token)
		}
	}
	return true
}

// Produce the VALUE(...,plain) token.
func ini_parser_fetch_value(parser *ini_parser_t) bool {
	if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
		return false
	}

	// Consume the token.
	start_mark := parser.mark
	skip(parser)
	end_mark := parser.mark
	token := ini_token_t{
		typ:        ini_VALUE_TOKEN,
		start_mark: start_mark,
		end_mark:   end_mark,
		value:      []byte("="),
	}
	ini_insert_token(parser, -1, &token)
    // Eat whitespaces.
    if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
        return false
    }
    for is_blank(parser.buffer, parser.buffer_pos) {
        skip(parser)
        if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
            return false
        }
    }
	// Produce the SCALAR(...,plain) token.
	if parser.buffer[parser.buffer_pos] == '\'' {
		// Is it a single-quoted scalar?
		if !ini_parser_scan_scalar(parser, &token, true) {
			return false
		}
		ini_insert_token(parser, -1, &token)
	} else if parser.buffer[parser.buffer_pos] == '"' {
		// Is it a double-quoted scalar?
		if !ini_parser_scan_scalar(parser, &token, false) {
			return false
		}
		ini_insert_token(parser, -1, &token)
	} else {
		// Is it a plain scalar?
		if !ini_parser_scan_plain_scalar(parser, &token) {
			return false
		}
		ini_insert_token(parser, -1, &token)
	}
	return true
}

// Scan a node value.
func ini_parser_scan_scalar(parser *ini_parser_t, token *ini_token_t, single bool) bool {
	start_mark := parser.mark
	var s []byte
	// Consume the content of the quoted scalar.
	for {
		if parser.unread < 2 && !ini_parser_update_buffer(parser, 2) {
			return false
		}
		if is_z(parser.buffer, parser.buffer_pos) {
			break
		}
		if single {
			if parser.buffer[parser.buffer_pos] == '\'' && parser.buffer[parser.buffer_pos+1] == '\'' {
				// Is is an escaped single quote.
				skip(parser)
				skip(parser)
			} else if parser.buffer[parser.buffer_pos] == '\'' {
				// It is a left single quote.
				skip(parser)
				// It is a non-escaped non-blank character.
				for parser.buffer[parser.buffer_pos] != '\'' {
					if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
						return false
					}
					if is_breakz(parser.buffer, parser.buffer_pos) {
						return false
					}
					s = read(parser, s)
				}
			}
			// Check if we are at the end of the scalar.
			if parser.buffer[parser.buffer_pos] == '\'' {
				skip(parser)
				break
			}
		} else {
			if parser.buffer[parser.buffer_pos] == '"' && parser.buffer[parser.buffer_pos+1] == '"' {
				// Is is an escaped double quote.
				skip(parser)
				skip(parser)
			} else if parser.buffer[parser.buffer_pos] == '"' {
				// It is a left double quote.
				skip(parser)
				// It is a non-escaped non-blank character.
				for parser.buffer[parser.buffer_pos] != '"' {
					if parser.unread < 1 || !ini_parser_update_buffer(parser, 1) {
						return false
					}
					if is_breakz(parser.buffer, parser.buffer_pos) {
						return false
					}
					s = read(parser, s)
				}
			} else if parser.buffer[parser.buffer_pos] == '\\' && is_break(parser.buffer, parser.buffer_pos+1) {
				// It is an escaped line break.
				if parser.unread < 3 && !ini_parser_update_buffer(parser, 3) {
					return false
				}
				skip(parser)
				skip_line(parser)
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
				ini_parser_set_scanner_error(parser, "while parsing a scalar",
					start_mark, "found unknown escape character")
			}
			// Check if we are at the end of the scalar.
			if parser.buffer[parser.buffer_pos] == '"' {
				skip(parser)
				break
			}
		}
		// Is it the end?
		if is_break(parser.buffer, parser.buffer_pos) {
			skip_line(parser)
			break
		}
	}
	// Eat the right quote.
	end_mark := parser.mark

	// Create a token.
	*token = ini_token_t{
		typ:        ini_SCALAR_TOKEN,
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

// Scan a plain scalar.
func ini_parser_scan_plain_scalar(parser *ini_parser_t, token *ini_token_t) bool {
	start_mark := parser.mark
	var s []byte
	// Consume the content of the plain scalar.
	for {
		if parser.unread < 1 && !ini_parser_update_buffer(parser, 1) {
			return false
		}
		if is_break(parser.buffer, parser.buffer_pos) {
			skip_line(parser)
			break
		}
		if is_z(parser.buffer, parser.buffer_pos) {
			break
		}
		if parser.buffer[parser.buffer_pos] == '=' {
			break
		}
		// Copy the character.
		s = read(parser, s)
	}
    end_mark := parser.mark
    // Trim blank characters.
    s = bytes.Trim(s, " ")
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
