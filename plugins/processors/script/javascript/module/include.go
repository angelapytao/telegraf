package module

import (
	// Register javascript modules.
	_ "github.com/influxdata/telegraf/plugins/processors/script/javascript/module/console"
	_ "github.com/influxdata/telegraf/plugins/processors/script/javascript/module/net"
	_ "github.com/influxdata/telegraf/plugins/processors/script/javascript/module/path"

	// _ "github.com/influxdata/telegraf/plugins/processors/script/javascript/module/processor"
	_ "github.com/influxdata/telegraf/plugins/processors/script/javascript/module/require"
	// _ "github.com/influxdata/telegraf/plugins/processors/script/javascript/module/windows"
)
