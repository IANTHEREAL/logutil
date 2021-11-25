package log_scanner

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

var (
	ErrNotFoundParser = errors.New("not suitable log parser")
	ErrLogIncomplete  = errors.New("incomplete log")
)

type logParserHub map[string]LogParser

var hub logParserHub

func registerLogParser(name string, l LogParser) {
	if hub == nil {
		return
	}

	hub[name] = l
}

func init() {
	hub = make(map[string]LogParser)
	registerLogParser("zap", &zapLogParser{})
}

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

func (l *LogScanner) Scan() (*Log, error) {
	line, next := l.reader.Scan()
	if !next {
		log.Printf("signatures %s", line)
		return nil, io.EOF
	}

	if l.parser == nil {
		if err := l.selectLogParser(line); err != nil {
			return nil, err
		}
	}

	lg, err := l.parser.Parse(line)
	if err == ErrLogIncomplete {
		return l.scanLog(line)
	} else if err != nil {
		return nil, err
	}

	lg.logPath = l.logPath
	return lg, nil
}

func (l *LogScanner) selectLogParser(content []byte) error {
	for name, parser := range hub {
		if parser.IsValid(content) {
			log.Printf("elect parser %s for log %s", name, l.logPath)
			l.parser = parser

			return nil
		}
	}

	return ErrNotFoundParser
}

func (l *LogScanner) scanLog(content []byte) (*Log, error) {
	rest, next := l.reader.Scan()
	if !next {
		return nil, io.EOF
	}

	content = append(content, rest...)
	log, err := l.parser.Parse(content)
	if err == ErrLogIncomplete {
		return l.scanLog(content)
	} else if err != nil {
		return nil, err
	}

	log.logPath = l.logPath
	return log, nil

}

type LogReader interface {
	Scan() ([]byte, bool)
	Close() error
}

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
	scanner.Buffer(buff, 102400)
	return &fileLogReader{
		logPath: logPath,
		fd:      fd,
		scanner: scanner,
		buff:    buff,
	}, nil
}

func (f *fileLogReader) Scan() ([]byte, bool) {
	next := f.scanner.Scan()
	if next {
		return f.scanner.Bytes(), true
	}

	log.Printf("error %+v", f.scanner.Err())

	return nil, false
}

func (f *fileLogReader) Close() error {
	return f.fd.Close()
}

type Log struct {
	logPath  string
	Time     []byte
	Level    []byte
	Position []byte
	Msg      []byte
}

func (l *Log) String() string {
	return fmt.Sprintf("%s %s %s %s %s", l.logPath, l.Position, l.Time, l.Level, l.Msg)
}

type LogParser interface {
	Parse(content []byte) (*Log, error)
	IsValid(content []byte) bool
}

type zapLogParser struct {
}

func newZapLogParser() LogParser {
	return &zapLogParser{}
}

func (z *zapLogParser) IsValid(content []byte) bool {
	_, err := z.Parse(content)
	return err == nil
}

func (z *zapLogParser) Parse(content []byte) (*Log, error) {
	zapLog := &ZapLog{Log: &Log{}}
	valid, _ := zapLog.Extract(content)
	if valid {
		return zapLog.Log, nil
	}

	return nil, ErrLogIncomplete
}

var constSpaceLsbrck = []byte(" [")

// filename: log.lde
type ZapLog struct {
	Rest []byte
	*Log
}

func (z *ZapLog) Extract(line []byte) (bool, error) {
	z.Rest = line
	var pos int

	// Checks if the rest starts with '[' and pass it
	if len(z.Rest) >= 1 && z.Rest[0] == '[' {
		z.Rest = z.Rest[1:]
	} else {
		return false, nil
	}

	// Take until ']' as Time(string)
	pos = bytes.IndexByte(z.Rest, ']')
	if pos >= 0 {
		z.Time = z.Rest[:pos]
		z.Rest = z.Rest[pos+1:]
	} else {
		return false, nil
	}

	// Checks if the rest starts with `" ["` and pass it
	if bytes.HasPrefix(z.Rest, constSpaceLsbrck) {
		z.Rest = z.Rest[len(constSpaceLsbrck):]
	} else {
		return false, nil
	}

	// Take until ']' as Level(string)
	pos = bytes.IndexByte(z.Rest, ']')
	if pos >= 0 {
		z.Level = z.Rest[:pos]
		z.Rest = z.Rest[pos+1:]
	} else {
		return false, nil
	}

	// Checks if the rest starts with `" ["` and pass it
	if bytes.HasPrefix(z.Rest, constSpaceLsbrck) {
		z.Rest = z.Rest[len(constSpaceLsbrck):]
	} else {
		return false, nil
	}

	// Take until ']' as Position(string)
	pos = bytes.IndexByte(z.Rest, ']')
	if pos >= 0 {
		z.Position = z.Rest[:pos]
		z.Rest = z.Rest[pos+1:]
	} else {
		return false, nil
	}

	// Checks if the rest starts with `" ["` and pass it
	if bytes.HasPrefix(z.Rest, constSpaceLsbrck) {
		z.Rest = z.Rest[len(constSpaceLsbrck):]
	} else {
		return false, nil
	}

	// Take until ']' as Msg(string)
	pos = bytes.IndexByte(z.Rest, ']')
	if pos >= 0 {
		z.Msg = z.Rest[:pos]
		z.Rest = z.Rest[pos+1:]
	} else {
		return false, nil
	}

	return true, nil
}
