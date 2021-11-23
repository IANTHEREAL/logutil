package log_extractor

type LogPatternOptions struct {
	Level []string
}

type LogFilter interface {
	Filter(pkgName, fnName string, opt *LogPatternOptions) bool
}

var logFilterMap map[string]LogFilter

func filterRegister(pkgPath string, filter LogFilter) {
	logFilterMap[pkgPath] = filter
}

func init() {
	logFilterMap = make(map[string]LogFilter)
	filterRegister("log", &officalLog{})
}

type FilterHub struct {
	options *LogPatternOptions
}

func NewFilterHub(opt *LogPatternOptions) *FilterHub {
	return &FilterHub{options: opt}
}

func (f *FilterHub) Filter(pkgName, fnName string) bool {
	for _, filter := range logFilterMap {
		if filter.Filter(pkgName, fnName, f.options) {
			return true
		}
	}

	return false
}

type officalLog struct{}

func (l *officalLog) Filter(pkgName, fnName string, opt *LogPatternOptions) bool {
	return pkgName == "log" && (fnName == "Printf" || fnName == "Print")
}
