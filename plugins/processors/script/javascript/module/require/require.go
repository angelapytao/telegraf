package require

import (
	"github.com/dop251/goja_nodejs/require"
	"github.com/pkg/errors"

	"github.com/influxdata/telegraf/plugins/processors/script/javascript"
)

func init() {
	javascript.AddSessionHook("require", func(s javascript.Session) {
		reg := require.NewRegistryWithLoader(loadSource)
		reg.Enable(s.Runtime())
	})
}

// loadSource disallows loading custom modules from file.
func loadSource(path string) ([]byte, error) {
	return nil, errors.Errorf("cannot load %v, only built-in modules are supported", path)
}
