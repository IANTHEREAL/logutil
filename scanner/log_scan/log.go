package scanner

import (
	"fmt"
	"log"
)

// Log is a structured representation of a line of log in the log file
type Log struct {
	LogPath  string
	Time     string
	Level    string
	Position string
	Msg      string
}

func (l *Log) String() string {
	return fmt.Sprintf("%s %s %s %s %s", l.LogPath, l.Position, l.Time, l.Level, l.Msg)
}

// LogScanner used to read and parse log from log files one line by one line
type LogScanner struct {
	logPath string

	reader LogReader
	parser LogParser
}

func NewLogScanner(logPath string) (*LogScanner, error) {
	reader, err := NewFileReader(logPath)
	if err != nil {
		return nil, err
	}

	return &LogScanner{
		logPath: logPath,
		reader:  reader,
	}, nil
}

// Scan return one line log
// If there are no more logs, it will regret EOF
func (l *LogScanner) Scan() (*Log, error) {
	line, err := l.reader.Scan()
	if err != nil {
		return nil, err
	}

	if l.parser == nil {
		if err := l.selectLogParser(line); err != nil {
			return nil, err
		}
	}

	lg, err := l.parser.Parse(line)

	for err == ErrLogIncomplete {
		// try to find a complete log
		lg, line, err = l.scanLog(line)
	}
	if err != nil {
		log.Printf("read log error %v: %s", err, line)
		return nil, err
	}

	lg.LogPath = l.logPath
	return lg, nil
}

// selectLogParser try to find the suitable log parser
func (l *LogScanner) selectLogParser(content []byte) error {
	for name, parser := range hub {
		if parser.IsSuitable(content) {
			log.Printf("elect parser %s for log %s", name, l.logPath)
			l.parser = parser

			return nil
		}
	}

	return ErrNotFoundParser
}

func (l *LogScanner) scanLog(content []byte) (*Log, []byte, error) {
	rest, err := l.reader.Scan()
	if err != nil {
		return nil, content, err
	}

	content = append(content, rest...)
	lg, err := l.parser.Parse(content)
	return lg, content, err
}

// GetLogPath returns the log path
func (l *LogScanner) GetLogPath() string {
	return l.logPath
}
