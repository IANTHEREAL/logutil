package log_extractor

import (
	"log"

	"github.com/IANTHEREAL/logutil/pkg/util"
	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
)

// LogPkgExtract designed to extract log printing level according to the log package used,
// e.g. error level using Errorf() method of the zap log package
type LogPkgExtract interface {
	Filter(pkgName, fnName, logMesage string) (string, bool)
}

var filterHub = make(map[string]LogPkgExtract)

func RegisterLogPkgFilter(pkg string, filter LogPkgExtract) {
	if _, exist := filterHub[pkg]; exist {
		log.Fatalf("log pattern filter hub for kind %s already exists", pkg)
	}
	filterHub[pkg] = filter
}

func init() {
	RegisterLogPkgFilter("log", &logPkg{})
	RegisterLogPkgFilter("zap", &zapLogPkg{})
}

// Filter used to determine whether the log pattern matched filter rule
type Filter struct {
	filterRule *logpattern_go_proto.LogPatternRule
}

func NewFilter(rule *logpattern_go_proto.LogPatternRule) *Filter {
	return &Filter{filterRule: rule}
}

// Filter used to log packaga/function name, and log format data to compute match result
func (f *Filter) Filter(pkgName, fnName, logMesage string) (string, bool) {
	for _, filter := range filterHub {
		if level, matched := filter.Filter(pkgName, fnName, logMesage); matched {
			if util.MatchLogPatternRule(f.filterRule, level, logMesage) {
				return level, true
			}
		}
	}

	return "", false
}

type logPkg struct{}

func (l *logPkg) Filter(pkgName, fnName, logMesage string) (string, bool) {
	level, matched := "", false

	if pkgName != "log" {
		return level, matched
	}

	if fnName == "ErrorFilterContextCanceled" || fnName == "Error" || fnName == "Errorf" {
		level, matched = "error", true
	} else if fnName == "Warnf" || fnName == "Warn" {
		level, matched = "warn", true
	} else if fnName == "Fatalf" || fnName == "Fatal" {
		level, matched = "fatal", true
	} else if fnName == "Printf" || fnName == "Print" || fnName == "Infof" || fnName == "Info" {
		level, matched = "info", true
	}

	return level, matched
}

type zapLogPkg struct{}

func (z *zapLogPkg) Filter(pkgName, fnName, logMesage string) (string, bool) {
	level, matched := "", false

	if pkgName != "zap" {
		return level, matched
	}

	if fnName == "Errorf" || fnName == "Error" {
		level, matched = "error", true
	} else if fnName == "Warnf" || fnName == "Warn" {
		level, matched = "warn", true
	} else if fnName == "Fatalf" || fnName == "Fatal" {
		level, matched = "fatal", true
	} else if fnName == "Infof" || fnName == "Info" {
		level, matched = "info", true
	}

	return level, matched
}
