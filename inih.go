package ini

import (
	"io"
)

const DEFAULT_SECTION = "default"

type ini_break_t int

// Line break types.
const (
	// Let the parser choose the break type.
	ini_ANY_BREAK ini_break_t = iota

	ini_CR_BREAK   // Use CR for line breaks (Mac style).
	ini_LN_BREAK   // Use LN for line breaks (Unix style).
	ini_CRLN_BREAK // Use CR LN for line breaks (DOS style).
)

type ini_error_type_t int

// Many bad things could happen with the parser and emitter.
const (
	// No error is produced.
	ini_NO_ERROR ini_error_type_t = iota

	ini_MEMORY_ERROR   // Cannot allocate or reallocate a block of memory.
	ini_READER_ERROR   // Cannot read or decode the input stream.
	ini_SCANNER_ERROR  // Cannot scan the input stream.
	ini_PARSER_ERROR   // Cannot parse the input stream.
	ini_COMPOSER_ERROR // Cannot compose a YAML document.
	ini_WRITER_ERROR   // Cannot write to the output stream.
	ini_EMITTER_ERROR  // Cannot emit a YAML stream.
)

// The pointer position.
type ini_mark_t struct {
	index  int // The position index.
	line   int // The position line.
	column int // The position column.
}

// Node Styles

type ini_style_t int8

type ini_scalar_style_t ini_style_t

// Scalar styles.
const (
	// Let the emitter choose the style.
	ini_ANY_SCALAR_STYLE ini_scalar_style_t = iota

	ini_PLAIN_SCALAR_STYLE         // The plain scalar style.
	ini_SINGLE_QUOTED_SCALAR_STYLE // The single-quoted scalar style.
	ini_DOUBLE_QUOTED_SCALAR_STYLE // The double-quoted scalar style.
)

// Tokens

type ini_token_type_t int

// Token types.
const (
	// An empty token.
	ini_NO_TOKEN ini_token_type_t = iota

	ini_DOCUMENT_START_TOKEN // A DOCUMENT-START token.
	ini_DOCUMENT_END_TOKEN   // A DOCUMENT-START token.

	ini_SECTION_START_TOKEN   // A SECTION-START token.
	ini_SECTION_INHERIT_TOKEN // A SECTION-INHERIT token.
    ini_SECTION_ENTRY_TOKEN // A SECTION-ENTRY token.

	ini_KEY_TOKEN // An VALUE token.
	ini_VALUE_TOKEN // An VALUE token.
	ini_SCALAR_TOKEN        // A SCALAR token.
	ini_MAP_TOKEN        // A MAP token.

	ini_COMMENT_START_TOKEN // A COMMENT-START token.
	ini_COMMENT_END_TOKEN   // A COMMENT-END token.
)

func (tt ini_token_type_t) String() string {
	switch tt {
	case ini_NO_TOKEN:
		return "ini_NO_TOKEN"
	case ini_DOCUMENT_START_TOKEN:
		return "ini_DOCUMENT_START_TOKEN"
	case ini_DOCUMENT_END_TOKEN:
		return "ini_DOCUMENT_END_TOKEN"
	case ini_SECTION_START_TOKEN:
		return "ini_SECTION_START_TOKEN"
	case ini_SECTION_INHERIT_TOKEN:
		return "ini_SECTION_INHERIT_TOKEN"
    case ini_SECTION_ENTRY_TOKEN:
        return "ini_SECTION_ENTRY_TOKEN"
	case ini_KEY_TOKEN:
		return "ini_KEY_TOKEN"
	case ini_VALUE_TOKEN:
		return "ini_VALUE_TOKEN"
	case ini_SCALAR_TOKEN:
		return "ini_SCALAR_TOKEN"
	case ini_COMMENT_START_TOKEN:
		return "ini_COMMENT_START_TOKEN"
	case ini_COMMENT_END_TOKEN:
		return "ini_COMMENT_END_TOKEN"
	}
	return "<unknown token>"
}

// The token structure.
type ini_token_t struct {
	// The token type.
	typ ini_token_type_t

	// The start/end of the token.
	start_mark, end_mark ini_mark_t

	// The scalar value
	// (for ini_SCALAR_TOKEN).
	value []byte

	// The scalar style (for ini_SCALAR_TOKEN).
	style ini_scalar_style_t
}

// Events

type ini_event_type_t int8

// Event types.
const (
	// An empty event.
	ini_NO_EVENT ini_event_type_t = iota

	ini_DOCUMENT_START_EVENT  // A DOCUMENT-START event.
	ini_DOCUMENT_END_EVENT    // A DOCUMENT-END event.
	ini_SECTION_INHERIT_EVENT // A SECTION-INHERIT event.
    ini_SECTION_ENTRY_EVENT   // A SECTION-ENTRY event.

    ini_MAPPING_EVENT  // An MAPPING event.
    ini_SCALAR_EVENT  // An SCALAR event.
	ini_COMMENT_EVENT // A COMMENT event.
)

// The event structure.
type ini_event_t struct {

	// The event type.
	typ ini_event_type_t

	// The start and end of the event.
	start_mark, end_mark ini_mark_t

	// The node value.
	value []byte

    // The tag (for ini_SCALAR_EVENT).
    tag []byte

	// The style (for ini_ELEMENT_START_EVENT).
	style ini_style_t
}

func (e *ini_event_t) event_type() string {
	switch e.typ {
	case ini_NO_EVENT:
		return "ini_NO_EVENT"
	case ini_DOCUMENT_START_EVENT:
		return "ini_DOCUMENT_START_EVENT"
	case ini_DOCUMENT_END_EVENT:
		return "ini_DOCUMENT_END_EVENT"
	case ini_SECTION_INHERIT_EVENT:
		return "ini_SECTION_INHERIT_EVENT"
    case ini_SECTION_ENTRY_EVENT:
        return "ini_SECTION_ENTRY_EVENT"
    case ini_MAPPING_EVENT:
        return "ini_MAPPING_EVENT"
	case ini_SCALAR_EVENT:
		return "ini_SCALAR_EVENT"
	case ini_COMMENT_EVENT:
		return "ini_COMMENT_EVENT"
	}
	return "<unknown token>"
}

func (e *ini_event_t) scalar_style() ini_scalar_style_t { return ini_scalar_style_t(e.style) }

// Nodes

const (
	ini_NULL_TAG   = "null"  // The tag 'null' with the only possible value: null.
	ini_BOOL_TAG   = "bool"  // The tag 'bool' with the values: true and false.
	ini_STR_TAG    = "str"   // The tag 'str' for string values.
	ini_INT_TAG    = "int"   // The tag 'int' for integer values.
	ini_FLOAT_TAG  = "float" // The tag 'float' for float values.
	ini_BINARY_TAG = "binary"
    ini_MAP_TAG = "map"
	
	ini_SECTION_TAG = "section"

    ini_DEFAULT_SCALAR_TAG   = ini_STR_TAG // The default scalar tag is str
)

// The prototype of a read handler.
//
// The read handler is called when the parser needs to read more bytes from the
// source. The handler should write not more than size bytes to the buffer.
// The number of written bytes should be set to the size_read variable.
//
// [in,out]   data        A pointer to an application data specified by
//                        ini_parser_set_input().
// [out]      buffer      The buffer to write the data from the source.
// [in]       size        The size of the buffer.
// [out]      size_read   The actual number of bytes read from the source.
//
// On success, the handler should return 1.  If the handler failed,
// the returned value should be 0. On EOF, the handler should set the
// size_read to 0 and return 1.
type ini_read_handler_t func(parser *ini_parser_t, buffer []byte) (n int, err error)

// The states of the parser.
type ini_parser_state_t int

const (
	ini_PARSE_DOCUMENT_START_STATE ini_parser_state_t = iota // Expect START.

	ini_PARSE_DOCUMENT_END_STATE        // Expect DOCUMENT-START.
	ini_PARSE_SECTION_FIRST_START_STATE // Expect SECTION-FIRST-ENTRY.
	ini_PARSE_SECTION_START_STATE       // Expect SECTION-ENTRY.
	ini_PARSE_SECTION_INHERIT_STATE 	// Expect SECTION-INHERIT.
	ini_PARSE_SECTION_ENTRY_STATE       // Expect SECTION-ENTRY.
	ini_PARSE_SECTION_KEY_STATE   // Expect a KEY.
    ini_PARSE_SECTION_VALUE_STATE   // Expect a VALUE.
	ini_PARSE_COMMENT_START_STATE       // Expect COMMENT-START.
	ini_PARSE_COMMENT_CONTENT_STATE     // Expect the content of a comment.
	ini_PARSE_COMMENT_END_STATE         // Expect COMMENT-END.
)

func (ps ini_parser_state_t) String() string {
	switch ps {
	case ini_PARSE_DOCUMENT_START_STATE:
		return "ini_PARSE_DOCUMENT_START_STATE"
	case ini_PARSE_DOCUMENT_END_STATE:
		return "ini_PARSE_DOCUMENT_END_STATE"
	case ini_PARSE_SECTION_FIRST_START_STATE:
		return "ini_PARSE_SECTION_FIRST_START_STATE"
	case ini_PARSE_SECTION_START_STATE:
		return "ini_PARSE_SECTION_START_STATE"
	case ini_PARSE_SECTION_INHERIT_STATE:
		return "ini_PARSE_SECTION_INHERIT_STATE"
	case ini_PARSE_SECTION_ENTRY_STATE:
		return "ini_PARSE_SECTION_ENTRY_STATE"
	case ini_PARSE_SECTION_KEY_STATE:
		return "ini_PARSE_SECTION_KEY_STATE"
	case ini_PARSE_SECTION_VALUE_STATE:
		return "ini_PARSE_SECTION_VALUE_STATE"
	case ini_PARSE_COMMENT_START_STATE:
		return "ini_PARSE_COMMENT_START_STATE"
	case ini_PARSE_COMMENT_CONTENT_STATE:
		return "ini_PARSE_COMMENT_CONTENT_STATE"
	case ini_PARSE_COMMENT_END_STATE:
		return "ini_PARSE_COMMENT_END_STATE"
	}
	return "<unknown parser state>"
}

// The parser structure.
//
// All members are internal. Manage the structure using the
// ini_parser_ family of functions.
type ini_parser_t struct {

	// Error handling
	error ini_error_type_t // Error type.

	problem string // Error description.

	// The byte about which the problem occured.
	problem_offset int
	problem_value  int
	problem_mark   ini_mark_t

	// The error context.
	context      string
	context_mark ini_mark_t

	// Reader stuff
	read_handler ini_read_handler_t // Read handler.

	input_file io.Reader // File input data.
	input      []byte    // String input data.
	input_pos  int

	eof bool // EOF flag

	buffer     []byte // The working buffer.
	buffer_pos int    // The current position of the buffer.

	unread int // The number of unread characters in the buffer.

	raw_buffer     []byte // The raw buffer.
	raw_buffer_pos int    // The current position of the buffer.

	offset int        // The offset of the current position (in bytes).
	mark   ini_mark_t // The mark of the current position.

    key_level int // The current key level.

	// Scanner stuff
	document_start_produced bool // Have we started to scan the input stream?
	document_end_produced   bool // Have we reached the end of the input stream?

	tokens          []ini_token_t // The tokens queue.
	tokens_head     int           // The head of the tokens queue.
	tokens_parsed   int           // The number of tokens fetched from the queue.
	token_available bool          // Does the tokens queue contain a token ready for dequeueing.

	// Parser stuff
	state  ini_parser_state_t   // The current parser state.
	states []ini_parser_state_t // The parser states stack.
	marks  []ini_mark_t         // The stack of marks.
}

// Emitter Definitions

// The prototype of a write handler.
//
// The write handler is called when the emitter needs to flush the accumulated
// characters to the output.  The handler should write @a size bytes of the
// @a buffer to the output.
//
// @param[in,out]   data        A pointer to an application data specified by
//                              ini_emitter_set_output().
// @param[in]       buffer      The buffer with bytes to be written.
// @param[in]       size        The size of the buffer.
//
// @returns On success, the handler should return @c 1.  If the handler failed,
// the returned value should be @c 0.
//
type ini_write_handler_t func(emitter *ini_emitter_t, buffer []byte) error

type ini_emitter_state_t int

// The emitter states.
const (
	// Expect DOCUMENT-START.
	ini_EMIT_DOCUMENT_START_STATE ini_emitter_state_t = iota

	ini_EMIT_DOCUMENT_END_STATE           // Expect DOCUMENT-END.
	ini_EMIT_FIRST_SECTION_START_STATE    // Expect the first section
	ini_EMIT_SECTION_START_STATE          // Expect the start of section.
	ini_EMIT_SECTION_FIRST_NODE_KEY_STATE // Expect the start of section.
	ini_EMIT_ELEMENT_KEY_STATE            // Expect the start of section.
	ini_EMIT_ELEMENT_VALUE_STATE          // Expect the node.
	ini_EMIT_SECTION_END_STATE            // Expect the end of section.
	ini_EMIT_COMMENT_START_STATE          // Expect the start of section.
	ini_EMIT_COMMENT_VALUE_STATE          // Expect the content of section.
	ini_EMIT_COMMENT_END_STATE            // Expect the end of section.
)

// The emitter structure.
//
// All members are internal.  Manage the structure using the @c ini_emitter_
// family of functions.
type ini_emitter_t struct {

	// Error handling

	error   ini_error_type_t // Error type.
	problem string           // Error description.

	// Writer stuff

	write_handler ini_write_handler_t // Write handler.

	output_buffer *[]byte   // String output data.
	output_file   io.Writer // File output data.

	buffer     []byte // The working buffer.
	buffer_pos int    // The current position of the buffer.

	raw_buffer     []byte // The raw buffer.
	raw_buffer_pos int    // The current position of the buffer.

	// Emitter stuff

	unicode    bool        // Allow unescaped non-ASCII characters?
	line_break ini_break_t // The preferred line break.

	state  ini_emitter_state_t   // The current emitter state.
	states []ini_emitter_state_t // The stack of states.

	events      []ini_event_t // The event queue.
	events_head int           // The head of the event queue.

	level int // The current flow level.

	root_context    bool // Is it the document root context?
	mapping_context bool // Is it a mapping context?

	line       int  // The current line.
	column     int  // The current column.
	whitespace bool // If the last character was a whitespace?
	open_ended bool // If an explicit document end is required?

	// Scalar analysis.
	scalar_data struct {
		value                 []byte             // The scalar value.
		multiline             bool               // Does the scalar contain line breaks?
		single_quoted_allowed bool               // Can the scalar be expressed in the single quoted style?
		style                 ini_scalar_style_t // The output style.
	}

	// Dumper stuff

	opened bool // If the document was already opened?
	closed bool // If the document was already closed?
}
