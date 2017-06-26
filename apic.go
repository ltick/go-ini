package ini

import (
	"io"
	"os"
)

func ini_insert_token(parser *ini_parser_t, pos int, token *ini_token_t) {
	//trace("ini_insert_token", "pos:", pos, "typ:", token.typ, "head:", parser.tokens_head, "len:", len(parser.tokens))

	// Check if we can move the queue at the beginning of the buffer.
	if parser.tokens_head > 0 && len(parser.tokens) == cap(parser.tokens) {
		if parser.tokens_head != len(parser.tokens) {
			copy(parser.tokens, parser.tokens[parser.tokens_head:])
		}
		parser.tokens = parser.tokens[:len(parser.tokens)-parser.tokens_head]
		parser.tokens_head = 0
	}
	parser.tokens = append(parser.tokens, *token)
	if pos < 0 {
		return
	}
	copy(parser.tokens[parser.tokens_head+pos+1:], parser.tokens[parser.tokens_head+pos:])
	parser.tokens[parser.tokens_head+pos] = *token
}

// Create a new parser object.
func ini_parser_initialize(parser *ini_parser_t) bool {
	*parser = ini_parser_t{
		raw_buffer: make([]byte, 0, input_raw_buffer_size),
		buffer:     make([]byte, 0, input_buffer_size),
	}
	return true
}

// Destroy a parser object.
func ini_parser_delete(parser *ini_parser_t) {
	*parser = ini_parser_t{}
}

// String read handler.
func ini_string_read_handler(parser *ini_parser_t, buffer []byte) (n int, err error) {
	if parser.input_pos == len(parser.input) {
		return 0, io.EOF
	}
	n = copy(buffer, parser.input[parser.input_pos:])
	parser.input_pos += n
	return n, nil
}

// File read handler.
func ini_file_read_handler(parser *ini_parser_t, buffer []byte) (n int, err error) {
	return parser.input_file.Read(buffer)
}

// Set a string input.
func ini_parser_set_input_string(parser *ini_parser_t, input []byte) {
	if parser.read_handler != nil {
		panic("must set the input source only once")
	}
	parser.read_handler = ini_string_read_handler
	parser.input = input
	parser.input_pos = 0
}

// Set a file input.
func ini_parser_set_input_file(parser *ini_parser_t, file *os.File) {
	if parser.read_handler != nil {
		panic("must set the input source only once")
	}
	parser.read_handler = ini_file_read_handler
	parser.input_file = file
}

// Create a new emitter object.
func ini_emitter_initialize(emitter *ini_emitter_t) bool {
	*emitter = ini_emitter_t{
		buffer:     make([]byte, output_buffer_size),
		raw_buffer: make([]byte, 0, output_raw_buffer_size),
		states:     make([]ini_emitter_state_t, 0, initial_stack_size),
		events:     make([]ini_event_t, 0, initial_queue_size),
	}
	return true
}

// Destroy an emitter object.
func ini_emitter_delete(emitter *ini_emitter_t) {
	*emitter = ini_emitter_t{}
}

// String write handler.
func ini_string_write_handler(emitter *ini_emitter_t, buffer []byte) error {
	*emitter.output_buffer = append(*emitter.output_buffer, buffer...)
	return nil
}

// File write handler.
func ini_file_write_handler(emitter *ini_emitter_t, buffer []byte) error {
	_, err := emitter.output_file.Write(buffer)
	return err
}

// Set a string output.
func ini_emitter_set_output_string(emitter *ini_emitter_t, output_buffer *[]byte) {
	if emitter.write_handler != nil {
		panic("must set the output target only once")
	}
	emitter.write_handler = ini_string_write_handler
	emitter.output_buffer = output_buffer
}

// Set a file output.
func ini_emitter_set_output_file(emitter *ini_emitter_t, file io.Writer) {
	if emitter.write_handler != nil {
		panic("must set the output target only once")
	}
	emitter.write_handler = ini_file_write_handler
	emitter.output_file = file
}

// Set if unescaped non-ASCII characters are allowed.
func ini_emitter_set_unicode(emitter *ini_emitter_t, unicode bool) {
	emitter.unicode = unicode
}

// Set the preferred line break character.
func ini_emitter_set_break(emitter *ini_emitter_t, line_break ini_break_t) {
	emitter.line_break = line_break
}

// Create DOCUMENT-START.
func ini_document_start_event_initialize(event *ini_event_t) bool {
	*event = ini_event_t{
		typ: ini_DOCUMENT_START_EVENT,
	}
	return true
}

// Create DOCUMENT-END.
func ini_document_end_event_initialize(event *ini_event_t) bool {
	*event = ini_event_t{
		typ: ini_DOCUMENT_END_EVENT,
	}
	return true
}

// Create ELEMENT.
func ini_scalar_event_initialize(event *ini_event_t, value []byte, style ini_scalar_style_t) bool {
	*event = ini_event_t{
		typ:     ini_SCALAR_EVENT,
		value:   value,
		style:   ini_style_t(style),
	}
	return true
}

// Destroy an event object.
func ini_event_delete(event *ini_event_t) {
	*event = ini_event_t{}
}
