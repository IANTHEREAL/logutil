package log_reporter

import (
	"io"
	"text/template"

	"github.com/IANTHEREAL/logutil/storage/keyvalue"
)

type Reporter struct {
	template string
	writer   io.Writer

	cov *Coverager
}

func NewReporter(store *keyvalue.Store, writer io.Writer, templatePath string) (*Reporter, error) {
	cov, err := NewCoverager(store)
	if err != nil {
		return nil, err
	}

	return &Reporter{
		cov:      cov,
		writer:   writer,
		template: templatePath,
	}, nil
}

func (r *Reporter) Output() error {
	tmp, err := template.ParseFiles(r.template)
	if err != nil {
		panic(err)
	}
	return tmp.Execute(r.writer, r.cov)
}
