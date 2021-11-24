package log_scanner

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
)

// 1. asterisk character (*, also called "star") matches zero or more characters,
//    for example, doc* matches doc and document but not dodo;
//    asterisk character must be the last character of wildcard word.
// 2. the question mark ? matches exactly one character
const (
	// asterisk [ * ]
	asterisk = '*'
	// question mark [ ? ]
	question = '?'
)

type patternTrie struct {
	sync.RWMutex

	root *node
}

type node struct {
	characters map[byte]item
	asterisk   item
	question   item
}

type item interface {
	child() *node
	setChild(*node)
	getLogPattern() []*logpattern_go_proto.LogPattern
	setLogPattern(*logpattern_go_proto.LogPattern)
}

type baseItem struct {
	ch *node

	pattern []*logpattern_go_proto.LogPattern
}

func (i *baseItem) child() *node {
	return i.ch
}

func (i *baseItem) setChild(c *node) {
	i.ch = c
}

func (i *baseItem) getLogPattern() []*logpattern_go_proto.LogPattern {
	return i.pattern
}

func (i *baseItem) setLogPattern(pattern *logpattern_go_proto.LogPattern) {
	/*for _, p := range i.pattern {
		if p.Pos == pattern.Pos {
			return
		}
	}*/
	if pattern.Signature[0] == "\"unit process error\"" {
		log.Printf("signatures %s %+v", pattern.Signature[0], pattern.Pos)
	}
	i.pattern = append(i.pattern, pattern)
}

func newNode() *node {
	return &node{characters: make(map[byte]item)}
}

// NewTrieSelector returns a trie Selector
func NewPatternTrie() *patternTrie {
	return &patternTrie{root: newNode()}
}

// Insert implements Selector's interface.
func (t *patternTrie) Insert(key string, pattern *logpattern_go_proto.LogPattern) error {
	if pattern == nil {
		return errors.New("Nil log pattern")
	}

	t.Lock()
	t.insert(t.root, key, pattern)
	t.Unlock()

	return nil
}

// if rule is nil, just extract nodes
func (t *patternTrie) insert(root *node, key string, pattern *logpattern_go_proto.LogPattern) bool {
	var (
		n      = root
		entity item
	)

	for i := 0; i < len(key); i++ {
		switch key[i] {
		case asterisk:
			entity = n.asterisk
		case question:
			entity = n.question
		default:
			entity = n.characters[key[i]]
		}
		if entity == nil {
			entity = &baseItem{}
			switch key[i] {
			case asterisk:
				n.asterisk = entity
			case question:
				n.question = entity
			default:
				n.characters[key[i]] = entity
			}
		}
		if entity.child() == nil {
			entity.setChild(newNode())
		}
		n = entity.child()
	}

	if entity != nil {
		entity.setLogPattern(pattern)
		return true
	}

	return false
}

// Match implements Selector's interface.
func (t *patternTrie) Match(key, level, position string) *MatchedResult {
	// find matched rules
	if t == nil {
		return nil
	}

	t.RLock()
	defer t.RUnlock()

	res := newMatchedResult(&MatchedOptions{
		LogLevel: level,
		Position: position,
	})

	//log.Printf("%s %s %s", key, level, position)

	t.matchNode(t.root, key, res)
	return res
}

func (t *patternTrie) matchNode(n *node, key string, res *MatchedResult) {
	if n == nil {
		return
	}

	var (
		ok     bool
		entity item
	)
	for i := range key {
		if n.asterisk != nil {
			res.append(n.asterisk)
			t.matchNode(n.asterisk.child(), key[i:], res)
		}

		if n.question != nil {
			if i == len(key)-1 {
				res.append(n.question)
			}

			t.matchNode(n.question.child(), key[i+1:], res)
		}

		entity, ok = n.characters[key[i]]
		if !ok {
			return
		}
		n = entity.child()
	}

	if entity != nil {
		res.append(entity)
	}

	if n.asterisk != nil {
		res.append(n.asterisk)
	}
}

type MatchedOptions struct {
	LogLevel string
	Position string
}

type MatchedResult struct {
	options  *MatchedOptions
	Patterns []*logpattern_go_proto.LogPattern
}

func newMatchedResult(options *MatchedOptions) *MatchedResult {
	return &MatchedResult{
		options: options,
	}
}

func (res *MatchedResult) empty() bool {
	return res == nil || len(res.Patterns) == 0
}

func (res *MatchedResult) append(entity item) {

	patterns := entity.getLogPattern()
	opt := res.options
	for _, pattern := range patterns {
		patternPos := pattern.GetPos()
		filePos := fmt.Sprintf("%s:%d", patternPos.FilePath, patternPos.LineNumber)

		if pattern != nil && opt != nil &&
			(opt.LogLevel == "" || strings.ToLower(opt.LogLevel) == strings.ToLower(pattern.Level)) &&
			(opt.Position == "" || strings.HasSuffix(filePos, opt.Position)) {
			res.Patterns = append(res.Patterns, pattern)
		}
	}
}
