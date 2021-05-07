package log

import (
	"github.com/influxdata/telegraf"
	"github.com/pkg/errors"
)

type serializer struct {
}

func NewSerializer() (*serializer, error) {
	s := &serializer{
	}
	return s, nil
}

func (s *serializer) Serialize(metric telegraf.Metric) ([]byte, error) {
	val,ok:= metric.GetField("message")
	if ok {
		return []byte(val.(string)), nil
	}
	return []byte(""),errors.New("找不到message字段")
}

func (s *serializer) SerializeBatch(metrics []telegraf.Metric) ([]byte, error) {
	value:=""
	 for _,m:=range metrics{
		 val,ok:= m.GetField("message") 
		 if ok {
			 value+=val.(string)
		 }else{
			 return []byte(""),errors.New("找不到message字段")
		 }
	 }
	return []byte(value), nil
}



