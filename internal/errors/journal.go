package apperrors

import "errors"

var (
	ErrWriterPoisoned  = errors.New("log writer is poisoned due to previous write/sync failure; recovery required")
	ErrCorruptRecord   = errors.New("corrupted record encountered in middle of file")
	ErrPayloadTooLarge = errors.New("payload length exceeds maximum allowed size")
)
