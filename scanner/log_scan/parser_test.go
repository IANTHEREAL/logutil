package scanner

import (
	"testing"

	. "github.com/pingcap/check"
)

func TestClient(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&testParserSuite{})

type testParserSuite struct {
}

func (t *testParserSuite) TestSelectParser(c *C) {
	var (
		parser         LogParser
		selectedParser string
	)
	contents, _ := testGenerateStandardZapLogs()
	for _, content := range contents {
		parser = nil
		for name, p := range hub {
			if p.IsSuitable([]byte(content)) {
				parser = p
				selectedParser = name
			}
		}
		c.Assert(parser, NotNil)
		c.Assert(selectedParser, Equals, "zap")
	}

	// TODO: it's a bug, need to fix later
	contents, errs := testGenerateSpecialZapLogs()
	for index, content := range contents {
		parser = nil
		for _, p := range hub {
			if p.IsSuitable([]byte(content)) {
				parser = p
			}
		}

		if errs[index] != nil {
			c.Assert(parser, IsNil)
		} else {
			c.Assert(parser, NotNil)
			c.Assert(selectedParser, Equals, "zap")
		}
	}

	contents, errs = testGenerateInValidZapLogs()
	for index, content := range contents {
		parser = nil
		for _, p := range hub {
			if p.IsSuitable([]byte(content)) {
				parser = p
			}
		}
		if errs[index] != nil {
			c.Assert(parser, IsNil)
		} else {
			c.Assert(parser, NotNil)
			c.Assert(selectedParser, Equals, "zap")
		}
	}
}

func (t *testParserSuite) TestParseZapLog(c *C) {
	parser := newZapLogParser()
	contents, logs := testGenerateStandardZapLogs()
	for i, content := range contents {
		lg, err := parser.Parse([]byte(content))
		c.Assert(err, IsNil)
		c.Assert(lg, DeepEquals, logs[i])
	}

	contents, errs := testGenerateSpecialZapLogs()
	for i, content := range contents {
		_, err := parser.Parse([]byte(content))
		c.Assert(err, DeepEquals, errs[i])
	}

	contents, errs = testGenerateInValidZapLogs()
	for i, content := range contents {
		_, err := parser.Parse([]byte(content))
		c.Assert(err, DeepEquals, errs[i])
	}
}

func testGenerateStandardZapLogs() ([]string, []*Log) {
	return []string{
			`[2021/11/18 23:20:53.596 +00:00] [INFO] [printer.go:54] ["Welcome to dm-worker"] ["Release Version"=v5.2.0-master] ["Git Commit Hash"=c91af794e65f54222b46094b287042cdadaf3bcb] ["Git Branch"=master] ["UTC Build Time"="2021-11-18 23:16:34"] ["Go Version"="go version go1.16.10 linux/amd64"]`,
			`[2021/11/18 23:20:53.596 +00:00] [INFO] [main.go:71] ["dm-worker config"="{\"name\":\"dm-worker-2\",\"log-level\":\"info\",\"log-file\":\"/log/dm-worker-2.log\",\"log-format\":\"text\",\"log-rotate\":\"\",\"join\":\"http://dm-master-0.dm-master.default:8261,http://dm-master-1.dm-master.default:8261,http://dm-master-2.dm-master.default:8261\",\"worker-addr\":\"0.0.0.0:8262\",\"advertise-addr\":\"dm-worker-2.dm-worker.default:8262\",\"config-file\":\"\",\"keepalive-ttl\":60,\"relay-keepalive-ttl\":1800,\"ssl-ca\":\"\",\"ssl-cert\":\"\",\"ssl-key\":\"\",\"cert-allowed-cn\":null}"]`,
			`[2021/11/18 23:21:56.901 +00:00] [ERROR] [source_worker.go:605] ["failed to update source status"] [component="worker controller"] [error="[code=11011:class=functional:scope=internal:level=high], Message: 0-1-7195 is not mysql GTID set"] [errorVerbose="[code=11011:class=functional:scope=internal:level=high], Message: 0-1-7195 is not mysql GTID set\ngithub.com/pingcap/ticdc/dm/pkg/terror.(*Error).Generate\n\tgithub.com/pingcap/ticdc/dm/pkg/terror/terror.go:267\ngithub.com/pingcap/ticdc/dm/pkg/gtid.(*MySQLGTIDSet).Set\n\tgithub.com/pingcap/ticdc/dm/pkg/gtid/gtid.go:122\ngithub.com/pingcap/ticdc/dm/pkg/binlog.(*Location).SetGTID\n\tgithub.com/pingcap/ticdc/dm/pkg/binlog/position.go:408\ngithub.com/pingcap/ticdc/dm/dm/worker.(*SourceWorker).updateSourceStatus\n\tgithub.com/pingcap/ticdc/dm/dm/worker/source_worker.go:251\ngithub.com/pingcap/ticdc/dm/dm/worker.(*SourceWorker).QueryStatus\n\tgithub.com/pingcap/ticdc/dm/dm/worker/source_worker.go:604\ngithub.com/pingcap/ticdc/dm/dm/worker.(*Server).QueryStatus\n\tgithub.com/pingcap/ticdc/dm/dm/worker/server.go:797\ngithub.com/pingcap/ticdc/dm/dm/pb._Worker_QueryStatus_Handler\n\tgithub.com/pingcap/ticdc/dm/dm/pb/dmworker.pb.go:2807\ngoogle.golang.org/grpc.(*Server).processUnaryRPC\n\tgoogle.golang.org/grpc@v1.40.0/server.go:1082\ngoogle.golang.org/grpc.(*Server).handleStream\n\tgoogle.golang.org/grpc@v1.40.0/server.go:1405\ngoogle.golang.org/grpc.(*Server).serveStreams.func1.1\n\tgoogle.golang.org/grpc@v1.40.0/server.go:746\nruntime.goexit\n\truntime/asm_amd64.s:1371"]`,
		}, []*Log{
			{Time: "2021/11/18 23:20:53.596 +00:00", Level: "INFO", Position: "printer.go:54", Msg: "\"Welcome to dm-worker\""},
			{Time: "2021/11/18 23:20:53.596 +00:00", Level: "INFO", Position: "main.go:71", Msg: `"dm-worker config"="{\"name\":\"dm-worker-2\",\"log-level\":\"info\",\"log-file\":\"/log/dm-worker-2.log\",\"log-format\":\"text\",\"log-rotate\":\"\",\"join\":\"http://dm-master-0.dm-master.default:8261,http://dm-master-1.dm-master.default:8261,http://dm-master-2.dm-master.default:8261\",\"worker-addr\":\"0.0.0.0:8262\",\"advertise-addr\":\"dm-worker-2.dm-worker.default:8262\",\"config-file\":\"\",\"keepalive-ttl\":60,\"relay-keepalive-ttl\":1800,\"ssl-ca\":\"\",\"ssl-cert\":\"\",\"ssl-key\":\"\",\"cert-allowed-cn\":null}"`},
			{Time: "2021/11/18 23:21:56.901 +00:00", Level: "ERROR", Position: "source_worker.go:605", Msg: "\"failed to update source status\""},
		}
}

func testGenerateSpecialZapLogs() ([]string, []error) {
	return []string{
		`[2021/11/18 23:20:53.596 +00:00] [INFO] [main.go:71] ["`,
		`dm-worker`,
		`config"="{\"name\":\"dm-worker-2\",\"log-level\":\"info\",\"log-file\":\"/log/dm-worker-2.log\",\"log-format\":\"text\",\"log-rotate\":\"\",\"join\":\"http://dm-master-0.dm-master.default:8261,http://dm-master-1.dm-master.default:8261,http://dm-master-2.dm-master.default:8261\",\"worker-addr\":\"0.0.0.0:8262\",\"advertise-addr\":\"dm-worker-2.dm-worker.default:8262\",\"config-file\":\"\",\"keepalive-ttl\":60,\"relay-keepalive-ttl\":1800,\"ssl-ca\":\"\",\"ssl-cert\":\"\",\"ssl-key\":\"\",\"cert-allowed-cn\":null}"][ xxx`,
		`xxxx][yyyyy`,
		`][zzzz]`,
		`[2021/11/18 23:21:56.901 +00:00] [ERROR] [source_worker.go:605] ["failed to update source status"] [component="worker controller"]`,
	}, []error{ErrLogIncomplete, ErrZapLog, ErrZapLog, ErrZapLog, ErrZapLog, nil}
}

func testGenerateInValidZapLogs() ([]string, []error) {
	return []string{
		`[2021/11/18 23:20:53.596 +00:00] [printer.go:54] [INFO] ["Welcome to dm-worker"] ["Release Version"=v5.2.0-master] ["Git Commit Hash"=c91af794e65f54222b46094b287042cdadaf3bcb] ["Git Branch"=master] ["UTC Build Time"="2021-11-18 23:16:34"] ["Go Version"="go version go1.16.10 linux/amd64"]`,
		`[2021/11/18 23:20:53.596 +00:00] [INFO] [printer.go:54] xxxx`,
		`[2021/11/18 23:20:53.596 +00:00] [INFO] xxx`,
		`[2021/11/18 23:20:53.596 +00:00] xxx`,
		`xxxxx`,
		`[2021/11/18 23:21:56.901 +00:00] [ERROR] [source_worker.go:605] ["failed to update source status"] [component="worker controller"]`,
	}, []error{ErrZapLog, ErrZapLog, ErrZapLog, ErrZapLog, ErrZapLog, nil}
}
