package journal

import (
	"bufio"
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func RecoverAndClean(path string) ([]RawEntry, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, &apperrors.RecoveryPipelineError{Step: apperrors.RecoverStepOpenFile, Offset: -1, Err: err}
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	stat, err := file.Stat()
	if err != nil {
		return nil, &apperrors.RecoveryPipelineError{Step: apperrors.RecoverStepStat, Offset: -1, Err: err}
	}
	fileSize := stat.Size()

	var entries []RawEntry
	headerBuf := make([]byte, headerSize)
	var validOffset int64

	for {
		_, err := io.ReadFull(reader, headerBuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, &apperrors.RecoveryPipelineError{Step: apperrors.RecoverStepReadHeader, Offset: validOffset, Err: err}
		}

		payloadLen := binary.BigEndian.Uint32(headerBuf[0:4])
		timestamp := int64(binary.BigEndian.Uint64(headerBuf[4:12]))
		storedHeaderCRC := binary.BigEndian.Uint32(headerBuf[12:16])
		storedPayloadCRC := binary.BigEndian.Uint32(headerBuf[16:20])

		computedHeaderCRC := crc32.ChecksumIEEE(headerBuf[0:12])
		if computedHeaderCRC != storedHeaderCRC {
			isZeroes, err := isZeroesToEOF(file, validOffset, fileSize)
			if err != nil {
				return nil, &apperrors.RecoveryPipelineError{Step: apperrors.RecoverStepCheckTail, Offset: validOffset, Err: err}
			}
			if isZeroes {
				break
			}
			return nil, &apperrors.RecoveryPipelineError{Step: apperrors.RecoverStepHeaderCRC, Offset: validOffset, Err: apperrors.ErrCorruptRecord}
		}

		if payloadLen > MaxPayloadSize {
			return nil, &apperrors.RecoveryPipelineError{Step: apperrors.RecoverStepPayloadSize, Offset: validOffset, Err: apperrors.ErrCorruptRecord}
		}

		recordEndOffset := validOffset + headerSize + int64(payloadLen)
		if recordEndOffset > fileSize {
			break
		}

		payload := make([]byte, payloadLen)
		_, err = io.ReadFull(reader, payload)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, &apperrors.RecoveryPipelineError{Step: apperrors.RecoverStepReadPayload, Offset: validOffset, Err: err}
		}

		computedPayloadCRC := crc32.ChecksumIEEE(payload)
		if computedPayloadCRC != storedPayloadCRC {
			if recordEndOffset < fileSize {
				return nil, &apperrors.RecoveryPipelineError{Step: apperrors.RecoverStepPayloadCRC, Offset: validOffset, Err: apperrors.ErrCorruptRecord}
			}
			break
		}

		validOffset = recordEndOffset
		entries = append(entries, RawEntry{Timestamp: timestamp, Payload: payload})
	}

	if validOffset == fileSize {
		return entries, nil
	}

	if err := file.Truncate(validOffset); err != nil {
		return nil, &apperrors.RecoveryPipelineError{Step: apperrors.RecoverStepTruncate, Offset: validOffset, Err: err}
	}

	if err := file.Sync(); err != nil {
		return nil, &apperrors.RecoveryPipelineError{Step: apperrors.RecoverStepSyncTruncate, Offset: validOffset, Err: err}
	}

	return entries, nil
}

func isZeroesToEOF(file *os.File, fromOffset int64, fileSize int64) (bool, error) {
	if fromOffset >= fileSize {
		return true, nil
	}

	buf := make([]byte, 4096)
	curr := fromOffset
	for curr < fileSize {
		toRead := int64(len(buf))
		if fileSize-curr < toRead {
			toRead = fileSize - curr
		}
		n, err := file.ReadAt(buf[:toRead], curr)
		if n == 0 {
			return false, nil
		}
		if err != nil && err != io.EOF {
			return false, err
		}
		for i := 0; i < n; i++ {
			if buf[i] != 0 {
				return false, nil
			}
		}
		curr += int64(n)
	}
	return true, nil
}
