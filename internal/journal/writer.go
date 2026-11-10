package journal

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"time"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

const (
	headerSize     = 20
	MaxPayloadSize = 16 * 1024 * 1024
)

type RawEntry struct {
	Timestamp int64
	Payload   []byte
}

type LogWriter struct {
	file     *os.File
	poisoned bool
}

func NewLogWriter(path string) (*LogWriter, error) {
	dir := filepath.Dir(path)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, &apperrors.WritePipelineError{Step: apperrors.StepOpenFile, Err: err}
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		_ = f.Close()
		return nil, &apperrors.WritePipelineError{Step: apperrors.StepOpenDir, Err: err}
	}

	syncErr := dirFile.Sync()
	closeErr := dirFile.Close()
	if syncErr != nil {
		_ = f.Close()
		return nil, &apperrors.WritePipelineError{Step: apperrors.StepSyncDir, Err: syncErr}
	}
	if closeErr != nil {
		_ = f.Close()
		return nil, &apperrors.WritePipelineError{Step: apperrors.StepCloseDirHandle, Err: closeErr}
	}

	return &LogWriter{file: f}, nil
}

func (lw *LogWriter) Append(payload []byte) error {
	if lw.poisoned {
		return apperrors.ErrWriterPoisoned
	}

	if len(payload) > MaxPayloadSize {
		return apperrors.ErrPayloadTooLarge
	}

	payloadLen := uint32(len(payload))
	timestamp := time.Now().UnixNano()

	buf := make([]byte, headerSize+len(payload))

	binary.BigEndian.PutUint32(buf[0:4], payloadLen)
	binary.BigEndian.PutUint64(buf[4:12], uint64(timestamp))

	headerCRC := crc32.ChecksumIEEE(buf[0:12])
	binary.BigEndian.PutUint32(buf[12:16], headerCRC)

	payloadCRC := crc32.ChecksumIEEE(payload)
	binary.BigEndian.PutUint32(buf[16:20], payloadCRC)

	copy(buf[headerSize:], payload)

	if _, err := lw.file.Write(buf); err != nil {
		lw.poisoned = true
		return &apperrors.WritePipelineError{Step: apperrors.StepWriteRecord, Err: err}
	}

	if err := lw.file.Sync(); err != nil {
		lw.poisoned = true
		return &apperrors.WritePipelineError{Step: apperrors.StepSyncRecord, Err: err}
	}

	return nil
}

func (lw *LogWriter) Close() error {
	return lw.file.Close()
}
