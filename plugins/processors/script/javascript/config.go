package javascript

import (
	"time"

	"github.com/pkg/errors"
)

// Config defines the Javascript source files to use for the processor.
type Config struct {
	Tag               string
	Source            string
	File              string
	Files             []string
	Params            map[string]interface{}
	Timeout           time.Duration
	TagOnException    string
	MaxCachedSessions int
}

// Validate returns an error if one (and only one) option is not set.
func (c Config) Validate() error {
	numConfigured := 0
	for _, set := range []bool{c.Source != "", c.File != "", len(c.Files) > 0} {
		if set {
			numConfigured++
		}
	}

	switch {
	case numConfigured == 0:
		return errors.Errorf("javascript must be defined via 'file', " +
			"'files', or inline as 'source'")
	case numConfigured > 1:
		return errors.Errorf("javascript can be defined in only one of " +
			"'file', 'files', or inline as 'source'")
	}

	return nil
}

func defaultConfig() Config {
	return Config{
		TagOnException:    "_js_exception",
		MaxCachedSessions: 4,
	}
}
