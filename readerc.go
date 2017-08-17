package ini

import (
	"io"
)

// Set the reader error and return 0.
func ini_parser_set_reader_error(parser *ini_parser_t, problem string, offset int, value int) bool {
	parser.error = ini_READER_ERROR
	parser.problem = problem
	parser.problem_offset = offset
	parser.problem_value = value
	return false
}

// Byte order marks.
const (
	bom_UTF8 = "\xef\xbb\xbf"
)

// Update the raw buffer.
func ini_parser_update_raw_buffer(parser *ini_parser_t) bool {
	size_read := 0

	// Return if the raw buffer is full.
	if parser.raw_buffer_pos == 0 && len(parser.raw_buffer) == cap(parser.raw_buffer) {
		return true
	}

	// Return on EOF.
	if parser.eof {
		return true
	}

	// Move the remaining bytes in the raw buffer to the beginning.
	if parser.raw_buffer_pos > 0 && parser.raw_buffer_pos < len(parser.raw_buffer) {
		copy(parser.raw_buffer, parser.raw_buffer[parser.raw_buffer_pos:])
	}
	parser.raw_buffer = parser.raw_buffer[:len(parser.raw_buffer)-parser.raw_buffer_pos]
	parser.raw_buffer_pos = 0

	// Call the read handler to fill the buffer.
	size_read, err := parser.read_handler(parser, parser.raw_buffer[len(parser.raw_buffer):cap(parser.raw_buffer)])
	parser.raw_buffer = parser.raw_buffer[:len(parser.raw_buffer)+size_read]
	if err == io.EOF {
		parser.eof = true
	} else if err != nil {
		return ini_parser_set_reader_error(parser, "input error: "+err.Error(), parser.offset, -1)
	}
	return true
}

// Ensure that the buffer contains at least `length` characters.
// Return true on success, false on failure.
//
// The length is supposed to be significantly less that the buffer size.
func ini_parser_update_buffer(parser *ini_parser_t, length int) bool {
	if parser.read_handler == nil {
		panic("read handler must be set")
	}

	// If the EOF flag is set and the raw buffer is empty, do nothing.
	if parser.eof && parser.raw_buffer_pos == len(parser.raw_buffer) {
		return true
	}

	// Return if the buffer contains enough characters.
	if parser.unread >= length {
		return true
	}

	// Move the unread characters to the beginning of the buffer.
	buffer_len := len(parser.buffer)
	if parser.buffer_pos > 0 && parser.buffer_pos < buffer_len {
		copy(parser.buffer, parser.buffer[parser.buffer_pos:])
		buffer_len -= parser.buffer_pos
		parser.buffer_pos = 0
	} else if parser.buffer_pos == buffer_len {
		buffer_len = 0
		parser.buffer_pos = 0
	}

	// Open the whole buffer for writing, and cut it before returning.
	parser.buffer = parser.buffer[:cap(parser.buffer)]

	// Fill the buffer until it has enough characters.
	first := true
	for parser.unread < length {

		// Fill the raw buffer if necessary.
		if !first || parser.raw_buffer_pos == len(parser.raw_buffer) {
			if !ini_parser_update_raw_buffer(parser) {
				parser.buffer = parser.buffer[:buffer_len]
				return false
			}
		}
		first = false

		// Decode the raw buffer.
	inner:
		for parser.raw_buffer_pos != len(parser.raw_buffer) {
			var value rune
			var width int

			raw_unread := len(parser.raw_buffer) - parser.raw_buffer_pos

			// Decode a UTF-8 character.  Check RFC 3629
			// (http://www.ietf.org/rfc/rfc3629.txt) for more details.
			//
			// The following table (taken from the RFC) is used for
			// decoding.
			//
			//    Char. number range |        UTF-8 octet sequence
			//      (hexadecimal)    |              (binary)
			//   --------------------+------------------------------------
			//   0000 0000-0000 007F | 0xxxxxxx
			//   0000 0080-0000 07FF | 110xxxxx 10xxxxxx
			//   0000 0800-0000 FFFF | 1110xxxx 10xxxxxx 10xxxxxx
			//   0001 0000-0010 FFFF | 11110xxx 10xxxxxx 10xxxxxx 10xxxxxx
			//
			// Additionally, the characters in the range 0xD800-0xDFFF
			// are prohibited as they are reserved for use with UTF-16
			// surrogate pairs.

			// Determine the length of the UTF-8 sequence.
			octet := parser.raw_buffer[parser.raw_buffer_pos]
			switch {
			case octet&0x80 == 0x00:
				width = 1
			case octet&0xE0 == 0xC0:
				width = 2
			case octet&0xF0 == 0xE0:
				width = 3
			case octet&0xF8 == 0xF0:
				width = 4
			default:
				// The leading octet is invalid.
				return ini_parser_set_reader_error(parser,
					"invalid leading UTF-8 octet",
					parser.offset, int(octet))
			}

			// Check if the raw buffer contains an incomplete character.
			if width > raw_unread {
				if parser.eof {
					return ini_parser_set_reader_error(parser,
						"incomplete UTF-8 octet sequence",
						parser.offset, -1)
				}
				break inner
			}

			// Decode the leading octet.
			switch {
			case octet&0x80 == 0x00:
				value = rune(octet & 0x7F)
			case octet&0xE0 == 0xC0:
				value = rune(octet & 0x1F)
			case octet&0xF0 == 0xE0:
				value = rune(octet & 0x0F)
			case octet&0xF8 == 0xF0:
				value = rune(octet & 0x07)
			default:
				value = 0
			}

			// Check and decode the trailing octets.
			for k := 1; k < width; k++ {
				octet = parser.raw_buffer[parser.raw_buffer_pos+k]

				// Check if the octet is valid.
				if (octet & 0xC0) != 0x80 {
					return ini_parser_set_reader_error(parser,
						"invalid trailing UTF-8 octet",
						parser.offset+k, int(octet))
				}

				// Decode the octet.
				value = (value << 6) + rune(octet&0x3F)
			}

			// Check the length of the sequence against the value.
			switch {
			case width == 1:
			case width == 2 && value >= 0x80:
			case width == 3 && value >= 0x800:
			case width == 4 && value >= 0x10000:
			default:
				return ini_parser_set_reader_error(parser,
					"invalid length of a UTF-8 sequence",
					parser.offset, -1)
			}

			// Check the range of the value.
			if value >= 0xD800 && value <= 0xDFFF || value > 0x10FFFF {
				return ini_parser_set_reader_error(parser,
					"invalid Unicode character",
					parser.offset, int(value))
			}

			// Check if the character is in the allowed range:
			//      #x9 | #xA | #xD | [#x20-#x7E]               (8 bit)
			//      | #x85 | [#xA0-#xD7FF] | [#xE000-#xFFFD]    (16 bit)
			//      | [#x10000-#x10FFFF]                        (32 bit)
			switch {
			case value == 0x09:
			case value == 0x0A:
			case value == 0x0D:
			case value >= 0x20 && value <= 0x7E:
			case value == 0x85:
			case value >= 0xA0 && value <= 0xD7FF:
			case value >= 0xE000 && value <= 0xFFFD:
			case value >= 0x10000 && value <= 0x10FFFF:
			default:
				return ini_parser_set_reader_error(parser,
					"control characters are not allowed",
					parser.offset, int(value))
			}

			// Move the raw pointers.
			parser.raw_buffer_pos += width
			parser.offset += width

			// Finally put the character into the buffer.
			if value <= 0x7F {
				// 0000 0000-0000 007F . 0xxxxxxx
				parser.buffer[buffer_len+0] = byte(value)
				buffer_len += 1
			} else if value <= 0x7FF {
				// 0000 0080-0000 07FF . 110xxxxx 10xxxxxx
				parser.buffer[buffer_len+0] = byte(0xC0 + (value >> 6))
				parser.buffer[buffer_len+1] = byte(0x80 + (value & 0x3F))
				buffer_len += 2
			} else if value <= 0xFFFF {
				// 0000 0800-0000 FFFF . 1110xxxx 10xxxxxx 10xxxxxx
				parser.buffer[buffer_len+0] = byte(0xE0 + (value >> 12))
				parser.buffer[buffer_len+1] = byte(0x80 + ((value >> 6) & 0x3F))
				parser.buffer[buffer_len+2] = byte(0x80 + (value & 0x3F))
				buffer_len += 3
			} else {
				// 0001 0000-0010 FFFF . 11110xxx 10xxxxxx 10xxxxxx 10xxxxxx
				parser.buffer[buffer_len+0] = byte(0xF0 + (value >> 18))
				parser.buffer[buffer_len+1] = byte(0x80 + ((value >> 12) & 0x3F))
				parser.buffer[buffer_len+2] = byte(0x80 + ((value >> 6) & 0x3F))
				parser.buffer[buffer_len+3] = byte(0x80 + (value & 0x3F))
				buffer_len += 4
			}

			parser.unread++
		}
		// On EOF, put NUL into the buffer and return.
		if parser.eof {
			parser.buffer[buffer_len] = 0
			buffer_len++
			parser.unread++
			break
		}
	}
	parser.buffer = parser.buffer[:buffer_len]
	return true
}
