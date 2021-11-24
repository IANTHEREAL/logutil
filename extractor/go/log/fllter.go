package log_extractor

import "log"

type LogPatternOptions struct {
	Level []string
}

type LogFilter interface {
	Filter(pkgName, fnName, logMesage string, opt *LogPatternOptions) (string, bool)
}

var logFilterMap map[string]LogFilter

func filterRegister(pkgPath string, filter LogFilter) {
	logFilterMap[pkgPath] = filter
}

func init() {
	logFilterMap = make(map[string]LogFilter)
	filterRegister("log", &officalLog{})
	filterRegister("zap", &zapLog{})
}

type FilterHub struct {
	options *LogPatternOptions
}

func NewFilterHub(opt *LogPatternOptions) *FilterHub {
	return &FilterHub{options: opt}
}

func (f *FilterHub) Filter(pkgName, fnName, logMesage string) (string, bool) {
	for _, filter := range logFilterMap {
		if level, isLog := filter.Filter(pkgName, fnName, logMesage, f.options); isLog {
			log.Printf("mateched log %s %s %s", pkgName, fnName, logMesage)
			return level, true
		}
	}

	return "", false
}

type officalLog struct{}

func (l *officalLog) Filter(pkgName, fnName, logMesage string, opt *LogPatternOptions) (string, bool) {
	if pkgName == "log" && fnName == "ErrorFilterContextCanceled" {
		return "error", true
	}

	return "", pkgName == "log" && (fnName == "Printf" || fnName == "Print")
}

type zapLog struct{}

func (z *zapLog) Filter(pkgName, fnName, logMesage string, opt *LogPatternOptions) (string, bool) {
	return "error", pkgName == "zap" && (fnName == "Errorf" || fnName == "Error" || fnName == "ErrorFilterContextCanceled")
}
