package module

import (
	// Register javascript modules.
	_ "github.com/influxdata/telegraf/plugins/processors/script/javascript/module/path"
	_ "github.com/influxdata/telegraf/plugins/processors/script/javascript/module/require"
)
