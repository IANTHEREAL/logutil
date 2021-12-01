package log_extractor

import "log"

// LogPatternRule is used to filter log printing pattern in the code,
// which is referred to as log pattern.
// The log pattern returned by filter.Filter() can definitely
// has the specified log level or signature (log keyword)
// usage:
// LogPatternRule.Level = ["error", "warn"] will filter error or warn level log print pattern
// LogPatternRule.Signature = ["network disconnect"] will filter log contains "network disconnect"
type LogPatternRule struct {
	Level     []string `toml:"log-level" json:"log-level"`
	Signature []string `toml:"log-signature" json:"log-signature"`
}

func (rule *LogPatternRule) Match(level string, message string) bool {
	if rule == nil {
		return true
	}

	// if there are no log level rule， return true；
	// otherwise return the matched result
	if len(rule.Level) > 0 {
		for _, l := range rule.Level {
			if l == level {
				return true
			}
		}
		return false
	}

	// TODO: add signatures matching algorithm

	return false
}

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
	filterRule *LogPatternRule
}

func NewFilter(rule *LogPatternRule) *Filter {
	if rule == nil {
		rule = &LogPatternRule{
			Level: []string{"error"},
		}
	}
	return &Filter{filterRule: rule}
}

// Filter used to log packaga/function name, and log format data to compute match result
func (f *Filter) Filter(pkgName, fnName, logMesage string) (string, bool) {
	for _, filter := range filterHub {
		if level, matched := filter.Filter(pkgName, fnName, logMesage); matched {
			if f.filterRule.Match(level, logMesage) {
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
