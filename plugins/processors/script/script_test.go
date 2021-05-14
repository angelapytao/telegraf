package script

import (
	"fmt"
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal/lib/common"
	"github.com/influxdata/telegraf/metric"
	"github.com/stretchr/testify/require"
)

func newM1() telegraf.Metric {
	m1, _ := metric.New("IIS_log",
		map[string]string{
			"verb":           "GET",
			"s-computername": "MIXEDCASE_hostname",
		},
		map[string]interface{}{
			"log.file.path": "/home/logs/trace-tracking-console.log",
			"log.offset":    "1234",
		},
		time.Now(),
		telegraf.Event,
	)
	return m1
}
func newM2() telegraf.Metric {
	fields := common.MapStr{
		"log": common.MapStr{
			"file": common.MapStr{
				"path": "/home/logs/trace-tracking-console.log",
			},
			"offset": 1234,
			"flags":  []string{"multiline"},
		},
	}

	m2, _ := metric.New("trace_log",
		map[string]string{},
		fields,
		time.Now(),
		telegraf.Event,
	)
	fmt.Printf("return fields: %v \n", m2.Fields())
	return m2
}

func TestMapStrToMapInterface(t *testing.T) {
	newM2()
}

func TestFieldConversions(t *testing.T) {
	const source = `
		function process(event) {
			if (event.Get("log") !== "") {
				var path = event.Get("log.file.path");
				var arr = path.split("/");
				var logName = arr[arr.length-1];
				var appName = logName.replace(/\.log.*/,"").replace("trace-","");
				appName = appName.replace(/[\-|\.]202\d{1}\-\d{2}\-\d{2}(\-\d{0,10})?/, "");
				event.Put("fields.logtopic", "trace-log-"+appName);
				event.Put("fields.evn", "dev");
				var logOffset = event.Get("log.offset");
				if (logOffset !== 0) {
					event.Put("offset", logOffset);
				}
			}
		}
	`
	tests := []struct {
		name   string
		plugin *Script
		check  func(t *testing.T, actual telegraf.Metric)
	}{
		{
			name: "Should add fields.logtopic field into metrics",
			plugin: &Script{
				Source: source,
			},
			check: func(t *testing.T, actual telegraf.Metric) {
				fv, ok := actual.GetField("fields.logtopic")
				require.True(t, ok)
				require.Equal(t, "trace-log-tracking-console", fv)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := tt.plugin.Apply(newM1())
			require.Len(t, metrics, 1)
			tt.check(t, metrics[0])
		})
	}
}

func TestMapStrFiledsConversions(t *testing.T) {
	const source = `
		function process(event) {
			if (event.Get("log") !== "") {
				var path = event.Get("log.file.path");
				var arr = path.split("/");
				var logName = arr[arr.length-1];
				var appName = logName.replace(/\.log.*/,"").replace("trace-","");
				appName = appName.replace(/[\-|\.]202\d{1}\-\d{2}\-\d{2}(\-\d{0,10})?/, "");
				event.Put("fields.logtopic", "trace-log-"+appName);
				// event.Put("fields.evn", "dev");
				// var logOffset = event.Get("log.offset");
				// if (logOffset !== 0) {
				// 	event.Put("offset", logOffset);
				// }
			}
		}
	`
	tests := []struct {
		name   string
		plugin *Script
		check  func(t *testing.T, actual telegraf.Metric)
	}{
		{
			name: "Should add fields.logtopic field into metrics",
			plugin: &Script{
				Source: source,
			},
			check: func(t *testing.T, actual telegraf.Metric) {
				t.Logf("actual events: %v \n", actual.Fields())
				fv, err := actual.GetFieldValue("fields.logtopic")
				t.Logf("fv: %v, err: %v \n", fv, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newM2()
			metrics := tt.plugin.Apply(m)
			require.Len(t, metrics, 1)
			tt.check(t, metrics[0])
		})
	}
}
