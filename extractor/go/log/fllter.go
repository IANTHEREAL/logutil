package log_extractor

import "log"

// LogPatternRule is used to filter log
// the log pattern returned by filter.Filter() can definitely
// match the specified log level or signature (log keyword)
// usage:
// LogPatternRule.Level = ["error"] will be filter ERROR level log
// LogPatternRule.Signature = ["network disconnect"] will be filter log contains ""network disconnect""
type LogPatternRule struct {
	Level     []string
	Signature []string
}

// LogPkgFilter designed to distinguish log levels according to the log package used,
// e.g. the log.Errorf log.Warn method of the zap log package
type LogPkgFilter interface {
	Filter(pkgName, fnName, logMesage string, opt *LogPatternRule) (string, bool)
}

var filterHub *FilterHub

func logPkgFilterRegister(pkgPath string, filter LogPkgFilter) {
	filterHub.logPkgFilterMap[pkgPath] = filter
}

func init() {
	filterHub = &FilterHub{
		logPkgFilterMap: make(map[string]LogPkgFilter),
	}
	logPkgFilterRegister("log", &officalLog{})
	logPkgFilterRegister("zap", &zapLog{})
}

// FilterHub accpets various log filtering methods, and try them one by one in
type FilterHub struct {
	filterRule      *LogPatternRule
	logPkgFilterMap map[string]LogPkgFilter
}

// Filter used to determine whether the log pattern matched filter rule
type Filter struct {
	filterRule *LogPatternRule
}

func NewFilter(rule *LogPatternRule) *Filter {
	return &Filter{filterRule: rule}
}

// Filter used to log packaga/function name, and log format data to compute match result
func (f *Filter) Filter(pkgName, fnName, logMesage string) (string, bool) {
	for _, filter := range filterHub.logPkgFilterMap {
		if level, isLog := filter.Filter(pkgName, fnName, logMesage, f.filterRule); isLog {
			log.Printf("mateched log %s %s %s", pkgName, fnName, logMesage)
			return level, true
		}
	}

	return "", false
}

type officalLog struct{}

func (l *officalLog) Filter(pkgName, fnName, logMesage string, opt *LogPatternRule) (string, bool) {
	if pkgName == "log" && fnName == "ErrorFilterContextCanceled" {
		return "error", true
	}

	return "", pkgName == "log" && (fnName == "Printf" || fnName == "Print")
}

type zapLog struct{}

func (z *zapLog) Filter(pkgName, fnName, logMesage string, opt *LogPatternRule) (string, bool) {
	return "error", pkgName == "zap" && (fnName == "Errorf" || fnName == "Error" || fnName == "ErrorFilterContextCanceled")
}
