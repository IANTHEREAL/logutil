package log_reporter

import (
	"io"
	"text/template"

	"github.com/IANTHEREAL/logutil/storage/keyvalue"
)

type Reporter struct {
	writer io.Writer

	cov *Coverager
}

// Report used print coverage data according to template format
func NewReporter(store *keyvalue.Store, writer io.Writer) (*Reporter, error) {
	cov, err := NewCoverager(store)
	if err != nil {
		return nil, err
	}

	return &Reporter{
		cov:    cov,
		writer: writer,
	}, nil
}

// Render outputs report to specified output
func (r *Reporter) Render(templatePath string, templateContent string) error {
	var (
		render *template.Template
		err    error
	)

	if len(templatePath) > 0 {
		render, err = template.ParseFiles(templatePath)
	} else {
		t := template.New("coverage")
		render, err = t.Parse(templateContent)
	}
	if err != nil {
		return err
	}

	return render.Execute(r.writer, r.cov)
}
