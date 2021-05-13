package javascript

import (
	"testing"
	"time"

	"github.com/influxdata/telegraf/internal/lib/common"
	"github.com/influxdata/telegraf/metric"
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
				var appName = logName.replace(".log","").replace("error-","");
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
		var fields common.MapStr
		metric, _ := metric.New("Beat", nil, fields, time.Now().UTC())
		metric.AddField("log.file.path", "/home/logs/trace-tracking-console.log")
		evt, err := p.Run(metric)
		t.Logf("err: %v, evt: %v", err, evt)
	})
}
