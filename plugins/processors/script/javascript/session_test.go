package javascript

import (
	"testing"
	"time"

	"github.com/influxdata/telegraf/internal/lib/common"
	"github.com/influxdata/telegraf/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionTestFunction(t *testing.T) {
	const source = `
		var fail = false;

		function register(params) {
			fail = params["fail"];
		}

		function process(event) {
			if (fail) {
				throw "intentional failure";
			}
			event.Put("hello", "world");

			if (event.Get("log") !== "") {
				var path = event.Get("log.file.path");
				var arr = path.split("/");
				var logName = arr[arr.length-1];
				var appName = logName.replace(".log","").replace("trace-","");
				appName = appName.replace(/[\-|\.]202\d{1}\-\d{2}\-\d{2}(\-\d{0,10})?/, "");
				event.Put("fields.logtopic", "trace-log-"+appName);
				event.Put("fields.evn", "dev");
				var logOffset = event.Get("log.offset");
				if (logOffset !== 0) {
				  event.Put("offset", logOffset);
				}
			}
			// return event;
		}
	`
	t.Run("test success", func(t *testing.T) {
		p, err := New(&Config{
			Source: source,
			Params: map[string]interface{}{
				"fail": false,
			},
		})

		fields := common.MapStr{
			"log": common.MapStr{
				"file": common.MapStr{
					"path": "/home/logs/trace-tracking-console.log",
				},
				"offset": 1234,
				"flags":  []string{"multiline"},
			},
		}
		metric, _ := metric.New("Beat", nil, fields, time.Now().UTC())
		evt, err := p.Run(metric)
		t.Logf("err: %v, evt: %v", err, evt)
	})
}

func TestMapStrFind(t *testing.T) {
	fields := common.MapStr{
		"log": common.MapStr{
			"file": common.MapStr{
				"path": "/home/logs/trace-tracking-console.log",
			},
			"offset": 1234,
			"flags":  []string{"multiline"},
		},
	}

	tests := []struct {
		name   string
		fields *common.MapStr
		check  func(t *testing.T)
	}{
		{
			name:   "test failure",
			fields: &fields,
			check: func(t *testing.T) {
				path, err := fields.GetValue("path")
				assert.Error(t, err, "key not found")
				assert.Nil(t, path)
			},
		},
		{
			name:   "test file.path success",
			fields: &fields,
			check: func(t *testing.T) {
				path, err := fields.GetValue("log.file.path")
				assert.NoError(t, err)
				require.Equal(t, "/home/logs/trace-tracking-console.log", path)
			},
		},
		{
			name:   "test offset success",
			fields: &fields,
			check: func(t *testing.T) {
				offset, err := fields.GetValue("log.offset")
				assert.NoError(t, err)
				require.Equal(t, 1234, offset)
			},
		},
		{
			name:   "test multiline success",
			fields: &fields,
			check: func(t *testing.T) {
				flags, err := fields.GetValue("log.flags")
				assert.NoError(t, err)
				require.Equal(t, []string{"multiline"}, flags)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t)
		})
	}
}
