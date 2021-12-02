package util

import (
	"context"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	proto "github.com/IANTHEREAL/logutil/proto"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
)

func MatchLogPatternRule(rule *proto.LogPatternRule, level string, message string) bool {
	if rule == nil {
		return true
	}

	// if there are no log level rule， return true；
	// otherwise return the matched result
	if len(rule.LogLevel) > 0 {
		for _, l := range rule.LogLevel {
			if strings.ToLower(l) == strings.ToLower(level) {
				return true
			}
		}
		return false
	}

	// TODO: add signatures matching algorithm
	// if message == "" skip it, sometimes we only match log level

	return true
}

// GetLogPatternRule returns log pattern rule from *keyvalue.Store
func GetLogPatternRule(store *keyvalue.Store) (*proto.LogPatternRule, error) {
	var rule *proto.LogPatternRule

	err := store.ScanLogPatternRule(context.Background(), func(_, value []byte) error {
		rule = &proto.LogPatternRule{}
		return rule.Unmarshal(value)
	})
	if err != nil {
		return nil, err
	}

	return rule, nil
}

// StrictDecodeFile decodes the toml file strictly. If any item in confFile file is not mapped
// into the Config struct, issue an error
func StrictDecodeFile(path string, cfg interface{}) error {
	metaData, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return err
	}

	if undecoded := metaData.Undecoded(); len(undecoded) > 0 {
		var undecodedItems []string
		for _, item := range undecoded {
			undecodedItems = append(undecodedItems, item.String())
		}
		err = fmt.Errorf("filter rule config file %s contained unknown configuration options: %s", path, strings.Join(undecodedItems, ", "))
	}

	return err
}
