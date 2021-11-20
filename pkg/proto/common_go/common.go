package commot_go

type Position struct {
	Line   int
	Column int
}

type XVName struct {
	Repo      string
	Package   string
	Path      string
	Pos       Position
	Signature string
}

type LogPattern struct {
	Name         XVName
	FunctionName string
	Singatures   []string
}
