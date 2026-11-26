package journal

import apperrors "github.com/sundayprincedev/ledger/internal/errors"

type FileJournal struct {
	path     string
	writer   *LogWriter
	poisoned bool
}

func Open(path string) *FileJournal {
	return &FileJournal{path: path}
}

func (j *FileJournal) Recover() ([]RawEntry, error) {
	return RecoverAndClean(j.path)
}

func (j *FileJournal) Append(payload []byte) error {
	if j.poisoned {
		return apperrors.ErrWriterPoisoned
	}
	if j.writer == nil {
		w, err := NewLogWriter(j.path)
		if err != nil {
			return err
		}
		j.writer = w
	}
	return j.writer.Append(payload)
}

func (j *FileJournal) Close() error {
	if j.writer == nil {
		return nil
	}
	return j.writer.Close()
}

func (j *FileJournal) Reset() {
	j.discardWriter()
	j.poisoned = false
}

func (j *FileJournal) Poison() {
	j.discardWriter()
	j.poisoned = true
}

func (j *FileJournal) discardWriter() {
	if j.writer == nil {
		return
	}
	_ = j.writer.Close()
	j.writer = nil
}
