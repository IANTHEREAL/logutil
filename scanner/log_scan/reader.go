package scanner

import (
	"bufio"
	"io"
	"os"
)

// LogReader defines the interface to read the log, which can read the log from different locations,
// such as disk files, log service
type LogReader interface {
	// Scan one line log content
	// If there are no more logs, it will regret EOF
	Scan() ([]byte, error)
	Close() error
}

// fileLogReader implements the LogReader interface, which can read the log from the log file on the disk
type fileLogReader struct {
	logPath string
	fd      *os.File
	scanner *bufio.Scanner

	buff []byte
}

func NewFileReader(logPath string) (LogReader, error) {
	fd, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}

	buff := make([]byte, 102400)

	scanner := bufio.NewScanner(fd)
	// Avoid size of single-line log is too large, resulting in reading errors
	scanner.Buffer(buff, 102400)
	return &fileLogReader{
		logPath: logPath,
		fd:      fd,
		scanner: scanner,
		buff:    buff,
	}, nil
}

func (f *fileLogReader) Scan() ([]byte, error) {
	next := f.scanner.Scan()
	if next {
		return f.scanner.Bytes(), f.scanner.Err()
	}

	if f.scanner.Err() != nil {
		return nil, f.scanner.Err()
	}

	return nil, io.EOF
}

func (f *fileLogReader) Close() error {
	return f.fd.Close()
}
