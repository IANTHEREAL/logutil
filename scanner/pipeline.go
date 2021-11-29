package log_scanner

import (
	"context"
	"io"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	scanner "github.com/IANTHEREAL/logutil/scanner/log_scan"
)

// The payload is the message transmission unit of the pipeline,
// which can carry log and log pattern
type Payload struct {
	log     *scanner.Log
	pattern *logpattern_go_proto.LogPattern
}

// Pipeline used pass messages between processing units (log scan, log match and record)
type Pipeline struct {
	ch chan *Payload
}

func NewPipiline(size int) *Pipeline {
	if size <= 0 {
		size = 1024
	}

	return &Pipeline{ch: make(chan *Payload, size)}
}

// Write writes payload in the pipeline
func (p *Pipeline) Write(ctx context.Context, payload *Payload) error {
	select {
	case <-ctx.Done():
	case p.ch <- payload:
	}

	return ctx.Err()
}

// Read return next payload and whether it's EOF
func (p *Pipeline) Read(ctx context.Context) (*Payload, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case payload, ok := <-p.ch:
		if !ok {
			return payload, io.EOF
		}

		return payload, nil
	}
}

// Close closes the pipeline
func (p *Pipeline) Close() {
	close(p.ch)
}
