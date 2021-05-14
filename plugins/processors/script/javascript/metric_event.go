package javascript

import (
	"time"

	"github.com/dop251/goja"
	"github.com/influxdata/telegraf"
	"github.com/pkg/errors"

	"github.com/influxdata/telegraf/internal/lib/common"
	"github.com/influxdata/telegraf/metric"
)

// IMPORTANT:
// This is the user facing API within Javascript processors. Do not make
// breaking changes to the JS methods. If you must make breaking changes then
// create a new version and require the user to specify an API version in their
// configuration (e.g. api_version: 2).

type metricEvent struct {
	vm        *goja.Runtime
	obj       *goja.Object
	inner     telegraf.Metric
	cancelled bool
}

func newMetircEvent(s Session) (Event, error) {
	e := &metricEvent{
		vm:  s.Runtime(),
		obj: s.Runtime().NewObject(),
	}
	e.init()
	return e, nil
}

func newMetircEventConstructor(s Session) func(call goja.ConstructorCall) *goja.Object {
	return func(call goja.ConstructorCall) *goja.Object {
		if len(call.Arguments) != 1 {
			panic(errors.New("Event constructor requires one argument"))
		}

		a0 := call.Argument(0).Export()

		var fields common.MapStr
		switch v := a0.(type) {
		case map[string]interface{}:
			fields = v
		case common.MapStr:
			fields = v
		default:
			panic(errors.Errorf("Event constructor requires a "+
				"map[string]interface{} argument but got %T", a0))
		}

		evt := &metricEvent{
			vm:  s.Runtime(),
			obj: call.This,
		}
		evt.init()

		// evt.reset(&beat.Event{Fields: fields})
		metric, _ := metric.New("BeatEvent", nil, fields, time.Now().UTC())
		evt.reset(metric)
		return nil
	}
}

func (e *metricEvent) init() {
	e.obj.Set("Get", e.get)
	e.obj.Set("Put", e.put)
	e.obj.Set("Cancel", e.cancel)
}

// reset the event so that it can be reused to wrap another event.
func (e *metricEvent) reset(b telegraf.Metric) error {
	e.inner = b
	e.cancelled = false
	e.obj.Set("_private", e)
	e.obj.Set("fields", e.vm.ToValue(e.inner.Fields))
	return nil
}

// Wrapped returns the wrapped beat.Event.
func (e *metricEvent) Wrapped() telegraf.Metric {
	return e.inner
}

// JSObject returns the goja.Value that represents the event within the
// Javascript runtime.
func (e *metricEvent) JSObject() goja.Value {
	return e.obj
}

// get returns the specified field. If the field does not exist then null is
// returned. If no field is specified then it returns entire object.
//
//	// javascript
// 	var dataset = evt.Get("event.dataset");
//
func (e *metricEvent) get(call goja.FunctionCall) goja.Value {
	a0 := call.Argument(0)
	if goja.IsUndefined(a0) {
		// event.Get() is the same as event.fields (but slower).
		return e.vm.ToValue(e.inner.Fields)
	}

	v, err := e.inner.GetFieldValue(a0.String())
	if err != nil {
		return goja.Null()
	}
	return e.vm.ToValue(v)
}

// put writes a value to the event. If there was a previous value assigned to
// the given field then the old object is returned. It throws an exception if
// you try to write a to a field where one of the intermediate values is not
// an object.
//
//	// javascript
// 	evt.Put("event.action", "process-created");
// 	evt.Put("geo.location", {"lon": -73.614830, "lat": 45.505918});
//
func (e *metricEvent) put(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 2 {
		panic(errors.New("Put requires two arguments (key and value)"))
	}

	key := call.Argument(0).String()
	value := call.Argument(1).Export()

	old, err := e.inner.PutFieldValue(key, value)
	if err != nil {
		panic(err)
	}
	return e.vm.ToValue(old)
}

// IsCancelled returns true if the event has been canceled.
func (e *metricEvent) IsCancelled() bool {
	return e.cancelled
}

// Cancel marks the event as cancelled. When the processor returns the event
// will be dropped.
func (e *metricEvent) Cancel() {
	e.cancelled = true
}

// cancel marks the event as cancelled.
func (e *metricEvent) cancel(call goja.FunctionCall) goja.Value {
	e.cancelled = true
	return goja.Undefined()
}
