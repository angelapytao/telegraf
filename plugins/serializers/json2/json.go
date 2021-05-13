package json2

import (
	"encoding/json"
	"github.com/influxdata/telegraf/plugins/common/store"
	jsoniter "github.com/json-iterator/go"
	"time"

	"github.com/influxdata/telegraf"
)

type serializer struct {
	TimestampUnits time.Duration
}

func NewSerializer(timestampUnits time.Duration) (*serializer, error) {
	s := &serializer{
		TimestampUnits: truncateDuration(timestampUnits),
	}
	return s, nil
}

func (s *serializer) Serialize(metric telegraf.Metric) ([]byte, error) {
	m := s.createObject(metric)
	serialized, err := json.Marshal(m)
	if err != nil {
		return []byte{}, err
	}
	serialized = append(serialized, '\n')

	return serialized, nil
}

func (s *serializer) SerializeBatch(metrics []telegraf.Metric) ([]byte, error) {
	objects := make([]interface{}, 0, len(metrics))
	for _, metric := range metrics {
		m := s.createObject(metric)
		objects = append(objects, m)
	}

	obj := map[string]interface{}{
		"metrics": objects,
	}

	serialized, err := json.Marshal(obj)
	if err != nil {
		return []byte{}, err
	}
	return serialized, nil
}

func (s *serializer) createObject(metric telegraf.Metric) map[string]interface{} {
	m := make(map[string]interface{})//, 4)
	m["tags"] = metric.Tags()
	//m["fields"] = metric.Fields()
	//仿照filebeat格式，发送到kafka,不显示fields字段名
	fields:= metric.Fields()
	for k,v:=range fields{
		//定制Log字段反序列化
		if  k=="log"{
			logDto:=new(store.LogDto)
			err:=jsoniter.Unmarshal([]byte(v.(string)),&logDto)
			if err==nil{
				m[k] =logDto
				continue
			}
		}
		m[k] =v
	}


	m["name"] = metric.Name()
	m["timestamp"] = metric.Time().UnixNano() / int64(s.TimestampUnits)
	return m
}

func truncateDuration(units time.Duration) time.Duration {
	// Default precision is 1s
	if units <= 0 {
		return time.Second
	}

	// Search for the power of ten less than the duration
	d := time.Nanosecond
	for {
		if d*10 > units {
			return d
		}
		d = d * 10
	}
}
