package all

import (
	_ "github.com/influxdata/telegraf/plugins/inputs/cpu"
	_ "github.com/influxdata/telegraf/plugins/inputs/disk"
	_ "github.com/influxdata/telegraf/plugins/inputs/diskio"
	_ "github.com/influxdata/telegraf/plugins/inputs/exec2"
	_ "github.com/influxdata/telegraf/plugins/inputs/execd"
	_ "github.com/influxdata/telegraf/plugins/inputs/http"
	_ "github.com/influxdata/telegraf/plugins/inputs/http_listener_v2"
	_ "github.com/influxdata/telegraf/plugins/inputs/internal"
	_ "github.com/influxdata/telegraf/plugins/inputs/linux_sysctl_fs"
	_ "github.com/influxdata/telegraf/plugins/inputs/log"
	_ "github.com/influxdata/telegraf/plugins/inputs/mem"
	_ "github.com/influxdata/telegraf/plugins/inputs/net"
	_ "github.com/influxdata/telegraf/plugins/inputs/nstat"
	_ "github.com/influxdata/telegraf/plugins/inputs/processes"
	_ "github.com/influxdata/telegraf/plugins/inputs/procstat2"
	_ "github.com/influxdata/telegraf/plugins/inputs/syslog"
	_ "github.com/influxdata/telegraf/plugins/inputs/sysstat"
	_ "github.com/influxdata/telegraf/plugins/inputs/system"
)
