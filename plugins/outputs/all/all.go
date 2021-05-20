package all

import (
	_ "github.com/influxdata/telegraf/plugins/inputs/exec2"
	_ "github.com/influxdata/telegraf/plugins/outputs/file"
	_ "github.com/influxdata/telegraf/plugins/outputs/http_notify"
	_ "github.com/influxdata/telegraf/plugins/outputs/kafka"
	_ "github.com/influxdata/telegraf/plugins/outputs/kafka2"
)
