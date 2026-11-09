package journal

type FileJournal struct {
	path   string
	writer *LogWriter
}

func Open(path string) *FileJournal {
	return &FileJournal{path: path}
}

func (j *FileJournal) Recover() ([]RawEntry, error) {
	return RecoverAndClean(j.path)
}

func (j *FileJournal) Append(payload []byte) error {
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
