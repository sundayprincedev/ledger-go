package journal

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func RecoverAndClean(path string) ([]RawEntry, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
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
			return nil, fmt.Errorf("read header at offset %d: %w", validOffset, err)
		}

		payloadLen := binary.BigEndian.Uint32(headerBuf[0:4])
		timestamp := int64(binary.BigEndian.Uint64(headerBuf[4:12]))
		storedHeaderCRC := binary.BigEndian.Uint32(headerBuf[12:16])
		storedPayloadCRC := binary.BigEndian.Uint32(headerBuf[16:20])

		computedHeaderCRC := crc32.ChecksumIEEE(headerBuf[0:12])
		if computedHeaderCRC != storedHeaderCRC {
			isZeroes, err := isZeroesToEOF(file, validOffset, fileSize)
			if err != nil {
				return nil, fmt.Errorf("check tail bytes at offset %d: %w", validOffset, err)
			}
			if isZeroes {
				break
			}
			return nil, fmt.Errorf("%w at offset %d: bad header CRC", apperrors.ErrCorruptRecord, validOffset)
		}

		if payloadLen > MaxPayloadSize {
			return nil, fmt.Errorf("%w at offset %d: payload size %d exceeds max cap", apperrors.ErrCorruptRecord, validOffset, payloadLen)
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
			return nil, fmt.Errorf("read payload at offset %d: %w", validOffset, err)
		}

		computedPayloadCRC := crc32.ChecksumIEEE(payload)
		if computedPayloadCRC != storedPayloadCRC {
			if recordEndOffset < fileSize {
				return nil, fmt.Errorf("%w at offset %d: payload checksum mismatch", apperrors.ErrCorruptRecord, validOffset)
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
		return nil, fmt.Errorf("truncate file: %w", err)
	}

	if err := file.Sync(); err != nil {
		return nil, fmt.Errorf("sync truncation to disk: %w", err)
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
