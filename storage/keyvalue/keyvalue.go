package keyvalue

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
)

// debug controls whether debugging information is emitted
const debug = false

// Size represents the size of data in bytes.
type Size uint64

func (s Size) Bytes() uint64 {
	return uint64(s)
}

// Common binary data sizes
const (
	Byte     Size = 1
	Kibibyte      = 1024 * Byte
	Mebibyte      = 1024 * Kibibyte
	Gibibyte      = 1024 * Mebibyte
	Tebibyte      = 1024 * Gibibyte
	Pebibyte      = 1024 * Tebibyte
)

// A Store implements the log pattern store interface for a keyvalue DB
type Store struct {
	db DB
}

// NewLogPatternStore returns a log pattern store backed by the given keyvalue DB.
func NewLogPatternStore(db DB) *Store {
	return &Store{db: db}
}

// Range is section of contiguous keys, including Start and excluding End.
type Range struct {
	Start, End []byte
}

// Options alters the behavior of an Iterator.
type Options struct{}

// A DB is a sorted key-value store with read/write access. DBs must be Closed
// when no longer used to ensure resources are not leaked.
type DB interface {
	// Get returns the value associated with the given key.  An io.EOF will be
	// returned if the key is not found.
	Get(context.Context, []byte, *Options) ([]byte, error)

	// ScanPrefix returns an Iterator for all key-values starting with the given
	// key prefix.  Options may be nil to use the defaults.
	ScanPrefix(context.Context, []byte, *Options) (Iterator, error)

	// ScanRange returns an Iterator for all key-values starting with the given
	// key range.  Options may be nil to use the defaults.
	ScanRange(context.Context, *Range, *Options) (Iterator, error)

	// Writer return a new write-access object
	Writer(context.Context) (Writer, error)

	// Close release the underlying resources for the database.
	Close(context.Context) error
}

// Iterator provides sequential access to a DB. Iterators must be Closed when
// no longer used to ensure that resources are not leaked.
type Iterator interface {
	io.Closer

	// Next returns the currently positioned key-value entry and moves to the next
	// entry. If there is no key-value entry to return, an io.EOF error is
	// returned.
	Next() (key, val []byte, err error)

	// Seeks positions the Iterator to the given key.  The key must be further
	// than the current Iterator's position.  If the key does not exist, the
	// Iterator is positioned at the next existing key.  If no such key exists,
	// io.EOF is returned.
	Seek(key []byte) error
}

// Writer provides write access to a DB. Writes must be Closed when no longer
// used to ensure that resources are not leaked.
type Writer interface {
	io.Closer

	// Write writes a key-value entry to the DB. Writes may be batched until the
	// Writer is Closed.
	Write(key, val []byte) error
}

// WritePool is a wrapper around a DB that automatically creates and flushes
// Writers as data size is written, creating a simple buffered interface for
// writing to a DB.  This interface is not thread-safe.
type WritePool struct {
	db   DB
	opts *PoolOptions

	wr     Writer
	writes int
	size   uint64
}

// PoolOptions is a set of options used by WritePools.
type PoolOptions struct {
	// MaxWrites is the number of calls to Write before the WritePool
	// automatically flushes the underlying Writer.  This defaults to 32000
	// writes.
	MaxWrites int

	// MaxSize is the total size of the keys and values given to Write before the
	// WritePool automatically flushes the underlying Writer.  This defaults to
	// 32MiB.
	MaxSize Size
}

func (o *PoolOptions) maxWrites() int {
	if o == nil || o.MaxWrites <= 0 {
		return 32000
	}
	return o.MaxWrites
}

func (o *PoolOptions) maxSize() uint64 {
	if o == nil || o.MaxSize <= 0 {
		return (Mebibyte * 32).Bytes()
	}
	return o.MaxSize.Bytes()
}

// NewPool returns a new WritePool for the given DB.  If opts==nil, its defaults
// are used.
func NewPool(db DB, opts *PoolOptions) *WritePool { return &WritePool{db: db, opts: opts} }

// Write buffers the given write until the pool becomes to large or Flush is
// called.
func (p *WritePool) Write(ctx context.Context, key, val []byte) error {
	if p.wr == nil {
		wr, err := p.db.Writer(ctx)
		if err != nil {
			return err
		}
		p.wr = wr
	}
	if err := p.wr.Write(key, val); err != nil {
		return err
	}
	p.size += uint64(len(key)) + uint64(len(val))
	p.writes++
	if p.opts.maxWrites() <= p.writes || p.opts.maxSize() <= p.size {
		return p.Flush()
	}
	return nil
}

// Flush ensures that all buffered writes are applied to the underlying DB.
func (p *WritePool) Flush() error {
	if p.wr == nil {
		return nil
	}
	if debug {
		log.Printf("Flushing (%d) %d", p.writes, Size(p.size))
	}
	err := p.wr.Close()
	p.wr = nil
	p.size, p.writes = 0, 0
	return err
}

// Write implements part of the GraphStore interface.
func (s *Store) Write(ctx context.Context, pattern *logpattern_go_proto.LogPattern) (err error) {
	wr, err := s.db.Writer(ctx)
	if err != nil {
		return fmt.Errorf("db writer error: %v", err)
	}
	defer func() {
		cErr := wr.Close()
		if err == nil && cErr != nil {
			err = fmt.Errorf("db writer close error: %v", cErr)
		}
	}()

	key, err := EncodeLogKey(pattern.Pos)
	if err != nil {
		return fmt.Errorf("encoding error: %v", err)
	}

	value, err := pattern.Marshal()
	if err != nil {
		return fmt.Errorf("encoding error: %v", err)
	}

	if err := wr.Write(key, value); err != nil {
		return fmt.Errorf("db write error: %v", err)
	}
	return nil
}

// Scan implements part of the graphstore.Service interface.
func (s *Store) Scan(ctx context.Context, fn func(key, value []byte) error) error {
	iter, err := s.db.ScanPrefix(ctx, logKeyPrefixBytes, &Options{})
	if err != nil {
		return fmt.Errorf("db seek error: %v", err)
	}
	defer iter.Close()
	for {
		key, val, err := iter.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("db iteration error: %v", err)
		}
		err = fn(key, val)
		if err != nil {
			return fmt.Errorf("invalid key/value entry: %v", err)
		}
	}
	return nil
}

// Close implements part of the graphstore.Service interface.
func (s *Store) Close(ctx context.Context) error { return s.db.Close(ctx) }

const (
	LogPatternKeyPrefix = "log:"
	FunctionKeyPrefix   = "fn:"
	CoverageKeyPrefix   = "cov:"
	// keySep is used to separate the repo, position of an encoded Entry key
	keySep    = '\n'
	keySepStr = string(keySep)
)

var (
	logKeyPrefixBytes      = []byte(LogPatternKeyPrefix)
	functionKeyPrefixBytes = []byte(FunctionKeyPrefix)
	coverageKeyPrefixBytes = []byte(CoverageKeyPrefix)
	keySepBytes            = []byte{keySep}
)

// EncodeLogKey returns a canonical encoding of  entity key
func EncodeLogKey(pos *logpattern_go_proto.Position) ([]byte, error) {
	if pos == nil {
		return nil, errors.New("invalid position: missing position for key encoding")
	}

	posBytes, err := pos.Marshal()
	if err != nil {
		return nil, err
	}

	return bytes.Join([][]byte{
		logKeyPrefixBytes,
		posBytes,
	}, nil), nil
}
