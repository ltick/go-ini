package ini

// Set the writer error and return false.
func ini_emitter_set_writer_error(emitter *ini_emitter_t, problem string) bool {
	emitter.error = ini_WRITER_ERROR
	emitter.problem = problem
	return false
}

// Flush the output buffer.
func ini_emitter_flush(emitter *ini_emitter_t) bool {
	if emitter.write_handler == nil {
		panic("write handler not set")
	}

	// Check if the buffer is empty.
	if emitter.buffer_pos == 0 {
		return true
	}

	if err := emitter.write_handler(emitter, emitter.buffer[:emitter.buffer_pos]); err != nil {
		return ini_emitter_set_writer_error(emitter, "write error: "+err.Error())
	}
	emitter.buffer_pos = 0
	return true
}
