package scanner

import (
	"io"

	. "github.com/pingcap/check"
)

var _ = Suite(&testLogSuite{})

type testLogSuite struct {
}

func (t *testLogSuite) TestRead(c *C) {
	logs, lgs := testGenerateStandardZapLogs()
	ls := &LogScanner{
		reader: newMockLogReader(logs),
	}

	for _, lg := range lgs {
		slg, err := ls.Scan()
		c.Assert(err, IsNil)
		c.Assert(slg, DeepEquals, lg)
	}

	logs, _ = testGenerateSpecialZapLogs()
	ls.reader = newMockLogReader(logs)

	slg, err := ls.Scan()
	c.Assert(err, IsNil)
	c.Assert(slg.Position, DeepEquals, "main.go:71")

	_, err = ls.Scan()
	c.Assert(err, Equals, io.EOF)
}

type mockLogReader struct {
	logs  []string
	index int
}

func (m *mockLogReader) Scan() ([]byte, error) {
	if m.index < len(m.logs) {
		lg := []byte(m.logs[m.index])
		m.index = m.index + 1
		return lg, nil
	}

	return nil, io.EOF
}

func (m *mockLogReader) Close() error {
	m.logs = m.logs[:0]
	return nil
}

func newMockLogReader(logs []string) LogReader {
	return &mockLogReader{
		logs: logs,
	}
}
