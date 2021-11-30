package matcher

import (
	"errors"
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
	getLogPattern() map[string]*BriefPattern
	setLogPattern(*logpattern_go_proto.LogPattern)
}

type baseItem struct {
	ch *node

	pattern map[string]*BriefPattern
}

func (i *baseItem) child() *node {
	return i.ch
}

func (i *baseItem) setChild(c *node) {
	i.ch = c
}

func (i *baseItem) getLogPattern() map[string]*BriefPattern {
	return i.pattern
}

func (i *baseItem) setLogPattern(pattern *logpattern_go_proto.LogPattern) {
	if i.pattern == nil {
		i.pattern = make(map[string]*BriefPattern)
	}
	bp := NewBriefPattern(pattern)
	i.pattern[bp.ID()] = bp
}

func newNode() *node {
	return &node{characters: make(map[byte]item)}
}

// NewPatternTrie returns a trie pattern matcher
func NewPatternTrie() *patternTrie {
	return &patternTrie{root: newNode()}
}

// Insert insers a pattern signature as key and the corresponding log pattern
func (t *patternTrie) Insert(key string, pattern *logpattern_go_proto.LogPattern) error {
	if pattern == nil {
		return errors.New("nil log pattern")
	}

	t.Lock()
	t.insert(t.root, key, pattern)
	t.Unlock()

	return nil
}

// insert builds a  trie, and the construction logic will replace the format symbol in the key, such as %s by asterisk(*)
func (t *patternTrie) insert(root *node, key string, pattern *logpattern_go_proto.LogPattern) bool {
	var (
		n      = root
		entity item
	)

	for i := 0; i < len(key); i++ {
		b := key[i]
		if key[i] == '%' {
			index, symbol := repalceFormatPlaceholder(key[i+1:])
			b = symbol
			i = i + index
		}

		switch b {
		case asterisk:
			entity = n.asterisk
		case question:
			entity = n.question
		default:
			entity = n.characters[b]
		}
		if entity == nil {
			entity = &baseItem{}
			switch b {
			case asterisk:
				n.asterisk = entity
			case question:
				n.question = entity
			default:
				n.characters[b] = entity
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

// Match looks for all matching patterns in trie and returns them in MatchResult.
// One log may match multiple patterns, which is related to the uniqueness of log data
func (t *patternTrie) Match(key, level, position string) *MatchedResult {
	if t == nil {
		return nil
	}

	t.RLock()
	defer t.RUnlock()

	res := newMatchedResult(&MatchedOptions{
		LogLevel: level,
		Position: position,
	})

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
			//log.Printf("%d %s %+v", i, key, n.asterisk.getLogPattern())
			res.append(n.asterisk)
			for index := i; index < len(key); index++ {
				t.matchNode(n.asterisk.child(), key[index:], res)
				t.matchNode(n.asterisk.child(), key[index+1:], res)
			}
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
	Patterns map[string]*BriefPattern
}

func newMatchedResult(options *MatchedOptions) *MatchedResult {
	return &MatchedResult{
		options:  options,
		Patterns: make(map[string]*BriefPattern),
	}
}

func (res *MatchedResult) empty() bool {
	return res == nil || len(res.Patterns) == 0
}

func (res *MatchedResult) append(entity item) {
	patterns := entity.getLogPattern()
	opt := res.options
	for _, pattern := range patterns {
		if opt != nil &&
			(opt.LogLevel == "" || strings.ToLower(opt.LogLevel) == strings.ToLower(pattern.matchedLevel)) &&
			(opt.Position == "" || strings.ToLower(opt.Position) == strings.ToLower(pattern.matchedPos)) {
			res.Patterns[pattern.ID()] = pattern
		}
	}
}

// repalceFormatPlaceholder replaces the format symbol in the key, such as %s by asterisk(*)
// `%`` Has been encountered before calling this function, `str` is characters after `%`
func repalceFormatPlaceholder(str string) (int, byte) {
	if len(str) == 0 {
		return 0, asterisk
	}

	if str[0] == '%' {
		return 1, '%'
	}

	for index, b := range str {
		if _, ok := formatPlaceholder[byte(b)]; ok {
			return index + 1, asterisk
		}
	}

	return len(str), asterisk
}

var formatPlaceholder = map[byte]struct{}{
	'v': {},
	'T': {},
	't': {},
	'b': {},
	'c': {},
	'd': {},
	'o': {},
	'O': {},
	'q': {},
	'x': {},
	'X': {},
	'U': {},
	'e': {},
	'E': {},
	'f': {},
	'F': {},
	'g': {},
	'G': {},
	's': {},
	'p': {},
}
