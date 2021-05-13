package processors

import "github.com/influxdata/telegraf"

type Creator func() telegraf.Processor

var Processors = map[string]Creator{}

func Add(name string, creator Creator) {
	Processors[name] = creator
}

type Processor interface {
	Run(event telegraf.Metric) (telegraf.Metric, error)
	String() string
}
