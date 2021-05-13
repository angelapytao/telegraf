package kafka2

import (
	"crypto/tls"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/influxdata/telegraf/plugins/common/store"

	"github.com/pkg/errors"

	"github.com/Shopify/sarama"
	"github.com/gofrs/uuid"
	"github.com/influxdata/telegraf"
	tlsint "github.com/influxdata/telegraf/internal/tls"
	"github.com/influxdata/telegraf/plugins/common/kafka"
	"github.com/influxdata/telegraf/plugins/outputs"
	"github.com/influxdata/telegraf/plugins/serializers"
)

var ValidTopicSuffixMethods = []string{
	"",
	"measurement",
	"tags",
}

var zeroTime = time.Unix(0, 0)

type (
	Kafka2 struct {
		//Brokers          []string    `toml:"brokers"`
		//Topic            string      `toml:"topic"`
		TopicTag         string      `toml:"topic_tag"`
		ExcludeTopicTag  bool        `toml:"exclude_topic_tag"`
		ClientID         string      `toml:"client_id"`
		TopicSuffix      TopicSuffix `toml:"topic_suffix"`
		RoutingTag       string      `toml:"routing_tag"`
		RoutingKey       string      `toml:"routing_key"`
		CompressionCodec int         `toml:"compression_codec"`
		RequiredAcks     int         `toml:"required_acks"`
		MaxRetry         int         `toml:"max_retry"`
		MaxMessageBytes  int         `toml:"max_message_bytes"`

		LogBrokers []LogBrokers `toml:"log_brokers"`

		Version string `toml:"version"`

		// Legacy TLS config options
		// TLS client certificate
		Certificate string
		// TLS client key
		Key string
		// TLS certificate authority
		CA string

		EnableTLS *bool `toml:"enable_tls"`
		tlsint.ClientConfig

		SASLUsername string `toml:"sasl_username"`
		SASLPassword string `toml:"sasl_password"`
		SASLVersion  *int   `toml:"sasl_version"`

		Log telegraf.Logger `toml:"-"`

		tlsConfig tls.Config

		producerFunc func(addrs []string, config *sarama.Config) (sarama.SyncProducer, error)
		//producer   sarama.SyncProducer
		producers map[string]sarama.SyncProducer
       // loker sync.Mutex
		serializer serializers.Serializer
		//*sarama.Config
		config *sarama.Config
	}
	TopicSuffix struct {
		Method    string   `toml:"method"`
		Keys      []string `toml:"keys"`
		Separator string   `toml:"separator"`
	}
	LogBrokers struct {
		Name                 string   `toml:"name"`                     //日志名称
		Brokers              []string `toml:"brokers"`                  //kafka服务器列表
		TopicFilterReg       string   `toml:"topic_filter_reg"`         //从message中提取topic的正则表达式
		TopicFilterNo        int      `toml:"topic_filter_no"`          //根据正则表达式提取了多个结果，第几个才是最终的topic
		TopicFilterDelPreSuf bool     `toml:"topic_filter_del_pre_suf"` //是否去掉提取的topic的首尾字符
	}
)

// DebugLogger logs messages from sarama at the debug level.
type DebugLogger struct {
}

func (*DebugLogger) Print(v ...interface{}) {
	args := make([]interface{}, 0, len(v)+1)
	args = append(args, "D! [sarama] ")
	log.Print(v...)
}

func (*DebugLogger) Printf(format string, v ...interface{}) {
	log.Printf("D! [sarama] "+format, v...)
}

func (*DebugLogger) Println(v ...interface{}) {
	args := make([]interface{}, 0, len(v)+1)
	args = append(args, "D! [sarama] ")
	log.Println(args...)
}

var sampleConfig = `
  ## URLs of kafka2 brokers
  brokers = ["localhost:9092"]
  ## Kafka2 topic for producer messages
  topic = "telegraf"

  ## The value of this tag will be used as the topic.  If not set the 'topic'
  ## option is used.
  # topic_tag = ""

  ## If true, the 'topic_tag' will be removed from to the metric.
  # exclude_topic_tag = false

  ## Optional Client id
  # client_id = "Telegraf"

  ## Set the minimal supported Kafka version.  Setting this enables the use of new
  ## Kafka features and APIs.  Of particular interest, lz4 compression
  ## requires at least version 0.10.0.0.
  ##   ex: version = "1.1.0"
  # version = ""

  ## Optional topic suffix configuration.
  ## If the section is omitted, no suffix is used.
  ## Following topic suffix methods are supported:
  ##   measurement - suffix equals to separator + measurement's name
  ##   tags        - suffix equals to separator + specified tags' values
  ##                 interleaved with separator

  ## Suffix equals to "_" + measurement name
  # [outputs.kafka.topic_suffix]
  #   method = "measurement"
  #   separator = "_"

  ## Suffix equals to "__" + measurement's "foo" tag value.
  ##   If there's no such a tag, suffix equals to an empty string
  # [outputs.kafka.topic_suffix]
  #   method = "tags"
  #   keys = ["foo"]
  #   separator = "__"

  ## Suffix equals to "_" + measurement's "foo" and "bar"
  ##   tag values, separated by "_". If there is no such tags,
  ##   their values treated as empty strings.
  # [outputs.kafka.topic_suffix]
  #   method = "tags"
  #   keys = ["foo", "bar"]
  #   separator = "_"

  ## The routing tag specifies a tagkey on the metric whose value is used as
  ## the message key.  The message key is used to determine which partition to
  ## send the message to.  This tag is prefered over the routing_key option.
  routing_tag = "host"

  ## The routing key is set as the message key and used to determine which
  ## partition to send the message to.  This value is only used when no
  ## routing_tag is set or as a fallback when the tag specified in routing tag
  ## is not found.
  ##
  ## If set to "random", a random value will be generated for each message.
  ##
  ## When unset, no message key is added and each message is routed to a random
  ## partition.
  ##
  ##   ex: routing_key = "random"
  ##       routing_key = "telegraf"
  # routing_key = ""

  ## CompressionCodec represents the various compression codecs recognized by
  ## Kafka in messages.
  ##  0 : No compression
  ##  1 : Gzip compression
  ##  2 : Snappy compression
  ##  3 : LZ4 compression
  # compression_codec = 0

  ##  RequiredAcks is used in Produce Requests to tell the broker how many
  ##  replica acknowledgements it must see before responding
  ##   0 : the producer never waits for an acknowledgement from the broker.
  ##       This option provides the lowest latency but the weakest durability
  ##       guarantees (some data will be lost when a server fails).
  ##   1 : the producer gets an acknowledgement after the leader replica has
  ##       received the data. This option provides better durability as the
  ##       client waits until the server acknowledges the request as successful
  ##       (only messages that were written to the now-dead leader but not yet
  ##       replicated will be lost).
  ##   -1: the producer gets an acknowledgement after all in-sync replicas have
  ##       received the data. This option provides the best durability, we
  ##       guarantee that no messages will be lost as long as at least one in
  ##       sync replica remains.
  # required_acks = -1

  ## The maximum number of times to retry sending a metric before failing
  ## until the next flush.
  # max_retry = 3

  ## The maximum permitted size of a message. Should be set equal to or
  ## smaller than the broker's 'message.max.bytes'.
  # max_message_bytes = 1000000

  ## Optional TLS Config
  # enable_tls = true
  # tls_ca = "/etc/telegraf/ca.pem"
  # tls_cert = "/etc/telegraf/cert.pem"
  # tls_key = "/etc/telegraf/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = false

  ## Optional SASL Config
  # sasl_username = "kafka"
  # sasl_password = "secret"

  ## SASL protocol version.  When connecting to Azure EventHub set to 0.
  # sasl_version = 1

  ## Data format to output.
  ## Each data format has its own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_OUTPUT.md
  # data_format = "influx"
`

func ValidateTopicSuffixMethod(method string) error {
	for _, validMethod := range ValidTopicSuffixMethods {
		if method == validMethod {
			return nil
		}
	}
	return fmt.Errorf("Unknown topic suffix method provided: %s", method)
}

func (k *Kafka2) GetTopicName(metric telegraf.Metric) (telegraf.Metric, string) {
	topic := ""
	if k.TopicTag != "" {
		if t, ok := metric.GetTag(k.TopicTag); ok {
			topic = t

			// If excluding the topic tag, a copy is required to avoid modifying
			// the metric buffer.
			if k.ExcludeTopicTag {
				metric = metric.Copy()
				metric.Accept()
				metric.RemoveTag(k.TopicTag)
			}
		}
	}

	var topicName string
	switch k.TopicSuffix.Method {
	case "measurement":
		topicName = topic + k.TopicSuffix.Separator + metric.Name()
	case "tags":
		var topicNameComponents []string
		topicNameComponents = append(topicNameComponents, topic)
		for _, tag := range k.TopicSuffix.Keys {
			tagValue := metric.Tags()[tag]
			if tagValue != "" {
				topicNameComponents = append(topicNameComponents, tagValue)
			}
		}
		topicName = strings.Join(topicNameComponents, k.TopicSuffix.Separator)
	default:
		topicName = topic
	}
	return metric, topicName
}

func (k *Kafka2) SetSerializer(serializer serializers.Serializer) {
	k.serializer = serializer
}

func (k *Kafka2) initConfig() error {
    config := sarama.NewConfig()
	if k.Version != "" {
		version, err := sarama.ParseKafkaVersion(k.Version)
		if err != nil {
			return err
		}
		config.Version = version
	}

	if k.ClientID != "" {
		config.ClientID = k.ClientID
	} else {
		config.ClientID = "Telegraf"
	}

	config.Producer.RequiredAcks = sarama.RequiredAcks(k.RequiredAcks)
	config.Producer.Compression = sarama.CompressionCodec(k.CompressionCodec)
	config.Producer.Retry.Max = k.MaxRetry
	config.Producer.Return.Successes = true

	if k.MaxMessageBytes > 0 {
		config.Producer.MaxMessageBytes = k.MaxMessageBytes
	}

	// Legacy support ssl config
	if k.Certificate != "" {
		k.TLSCert = k.Certificate
		k.TLSCA = k.CA
		k.TLSKey = k.Key
	}

	if k.EnableTLS != nil && *k.EnableTLS {
		config.Net.TLS.Enable = true
	}

	tlsConfig, err := k.ClientConfig.TLSConfig()
	if err != nil {
		return err
	}

	if tlsConfig != nil {
		config.Net.TLS.Config = tlsConfig

		// To maintain backwards compatibility, if the enable_tls option is not
		// set TLS is enabled if a non-default TLS config is used.
		if k.EnableTLS == nil {
			k.Log.Warnf("Use of deprecated configuration: enable_tls should be set when using TLS")
			config.Net.TLS.Enable = true
		}
	}

	if k.SASLUsername != "" && k.SASLPassword != "" {
		config.Net.SASL.User = k.SASLUsername
		config.Net.SASL.Password = k.SASLPassword
		config.Net.SASL.Enable = true

		version, err := kafka.SASLVersion(config.Version, k.SASLVersion)
		if err != nil {
			return err
		}
		config.Net.SASL.Version = version
	}
	k.config=config
	return nil
}

func (k *Kafka2) Connect() error {
	err := ValidateTopicSuffixMethod(k.TopicSuffix.Method)
	if err != nil {
		return err
	}

	//config := sarama.NewConfig()
	//
	//if k.Version != "" {
	//	version, err := sarama.ParseKafkaVersion(k.Version)
	//	if err != nil {
	//		return err
	//	}
	//	config.Version = version
	//}
	//
	//if k.ClientID != "" {
	//	config.ClientID = k.ClientID
	//} else {
	//	config.ClientID = "Telegraf"
	//}
	//
	//config.Producer.RequiredAcks = sarama.RequiredAcks(k.RequiredAcks)
	//config.Producer.Compression = sarama.CompressionCodec(k.CompressionCodec)
	//config.Producer.Retry.Max = k.MaxRetry
	//config.Producer.Return.Successes = true
	//
	//if k.MaxMessageBytes > 0 {
	//	config.Producer.MaxMessageBytes = k.MaxMessageBytes
	//}
	//
	//// Legacy support ssl config
	//if k.Certificate != "" {
	//	k.TLSCert = k.Certificate
	//	k.TLSCA = k.CA
	//	k.TLSKey = k.Key
	//}
	//
	//if k.EnableTLS != nil && *k.EnableTLS {
	//	config.Net.TLS.Enable = true
	//}
	//
	//tlsConfig, err := k.ClientConfig.TLSConfig()
	//if err != nil {
	//	return err
	//}
	//
	//if tlsConfig != nil {
	//	config.Net.TLS.Config = tlsConfig
	//
	//	// To maintain backwards compatibility, if the enable_tls option is not
	//	// set TLS is enabled if a non-default TLS config is used.
	//	if k.EnableTLS == nil {
	//		k.Log.Warnf("Use of deprecated configuration: enable_tls should be set when using TLS")
	//		config.Net.TLS.Enable = true
	//	}
	//}
	//
	//if k.SASLUsername != "" && k.SASLPassword != "" {
	//	config.Net.SASL.User = k.SASLUsername
	//	config.Net.SASL.Password = k.SASLPassword
	//	config.Net.SASL.Enable = true
	//
	//	version, err := kafka.SASLVersion(config.Version, k.SASLVersion)
	//	if err != nil {
	//		return err
	//	}
	//	config.Net.SASL.Version = version
	//}

	err =k.initConfig()
	if err!=nil{
		return err
	}
	return k.createProducer()
}

func (k *Kafka2) Close() error {
	//return k.producer.Close()
	//k.loker.Lock()
	//defer  k.loker.Unlock()

	for _, p := range k.producers {
		err := p.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *Kafka2) SampleConfig() string {
	return sampleConfig
}

func (k *Kafka2) Description() string {
	return "Configuration for the Kafka2 server to send metrics to"
}

func (k *Kafka2) routingKey(metric telegraf.Metric) (string, error) {
	if k.RoutingTag != "" {
		key, ok := metric.GetTag(k.RoutingTag)
		if ok {
			return key, nil
		}
	}

	if k.RoutingKey == "random" {
		u, err := uuid.NewV4()
		if err != nil {
			return "", err
		}
		return u.String(), nil
	}

	return k.RoutingKey, nil
}

func (k *Kafka2) Write(metrics []telegraf.Metric) error {
	for _, metric := range metrics {
		//metric, topic := k.GetTopicName(metric)

		buf, err := k.serializer.Serialize(metric)
		if err != nil {
			k.Log.Debugf("Could not serialize metric: %v", err)
			continue
		}
		_logName, ok := metric.GetField("log_name")
		if !ok {
			return errors.New("log_name为空！")
		}
		_logDto , ok :=metric.GetField("log")
		if !ok {
			return errors.New("log为空！")
		}

		logDto:=new(store.LogDto)
		err=jsoniter.Unmarshal([]byte(_logDto.(string)),&logDto)
		if err != nil {
			return errors.New("log 反序列化失败！")
		}
		fileName:=logDto.File.Path
		offset:=logDto.Offset

		topic := ""
		var hosts []string
		logName := _logName.(string)
		_topic, ok := metric.GetField("topic")
		if ok {
			topic = _topic.(string)
			if topic == "" {
				return errors.New("metric中的topic字段为空！")
			}
			_hosts, ok := metric.GetField("hosts")
			if ok {
				hosts = _hosts.([]string)
			}
		} else {
			//如果metric中缺少topic字段，则根据配置文件从message中提取
			topic, err = k.fetchTopic(logName, string(buf))
			if err != nil {
				return errors.New("metric中缺少topic字段！" + err.Error())
			}
			if topic == "" {
				return errors.New("metric中缺少topic字段，并且在message中提取topic失败！")
			}
		}

		m := &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.ByteEncoder(buf),
		}

		// Negative timestamps are not allowed by the Kafka protocol.
		if !metric.Time().Before(zeroTime) {
			m.Timestamp = metric.Time()
		}

		key, err := k.routingKey(metric)
		if err != nil {
			return fmt.Errorf("could not generate routing key: %v", err)
		}

		if key != "" {
			m.Key = sarama.StringEncoder(key)
		}

		producer:=k.getProducer(logName,topic,hosts)
		if producer==nil{
			return fmt.Errorf("%s,%s,%v 连接kafka失败", logName,topic,hosts)
		}
		_, _, prodErr := producer.SendMessage(m)

		fmt.Println("send------------", string(buf), offset)
		if prodErr != nil {
			errP := prodErr.(*sarama.ProducerError)
			if errP.Err == sarama.ErrMessageSizeTooLarge {
				k.Log.Error("Message too large, consider increasing `max_message_bytes`; dropping batch")
				return nil
			}
			if errP.Err == sarama.ErrInvalidTimestamp {
				k.Log.Error("The timestamp of the message is out of acceptable range, consider increasing broker `message.timestamp.difference.max.ms`; dropping batch")
				return nil
			}
			return errP
		}
		logOffsetDto, ok := store.MapLogOffset[fileName]
		if !ok {
			logOffsetDto = new(store.LogOffset)
			logOffsetDto.FileName = fileName + ".offset"
			store.MapLogOffset[fileName] = logOffsetDto
		}
		err = logOffsetDto.Set(offset )
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *Kafka2) createProducer() error {
	k.producers = make(map[string]sarama.SyncProducer)
	//k.loker.Lock()
	//defer k.loker.Unlock()

	for _, v := range k.LogBrokers {
		producer, err := k.producerFunc(v.Brokers, k.config)
		if err != nil {
			return err
		}
		k.producers[v.Name] = producer
	}

	return nil
}

func (k *Kafka2) fetchTopic(name, msg string) (string, error) {
	b := new(LogBrokers)
	for _, l := range k.LogBrokers {
		if l.Name == name {
			b = &l
			break
		}
	}
	if b == nil {
		return "", errors.New(name + "没有匹配的配置节点，请检查[[outputs.kafka2.log_brokers]]")
	}
	if b.TopicFilterReg == "" {
		return b.Name, nil
	}
	reg := regexp.MustCompile(b.TopicFilterReg)
	r := reg.FindAllString(msg, -1)
	if b.TopicFilterNo >= len(r) {
		return "", errors.New("解析message中的topic出错，索引超出数组界限！请检查配置参数：topic_filter_reg和topic_filter_no")
	}
	topic := r[b.TopicFilterNo]
	if b.TopicFilterDelPreSuf {
		topic = topic[1:]
		topic = topic[0 : len(topic)-1]
	}
	return topic, nil
}

func (k *Kafka2)getProducer(key,topic string,hosts []string)sarama.SyncProducer{
	//k.loker.Lock()
	//defer k.loker.Unlock()
	if topic!="" && len(hosts)>0 {
		key =strings.Join(hosts,"")+"_"+topic
		p,isok:= k.producers[key]
		if  isok{
			return p
		}
		producer, err := k.producerFunc(hosts, k.config)
		if err != nil {
			fmt.Println("连接kafka出错",hosts,topic)
			return nil
		}
		k.producers[key] = producer
		return producer
	}

	p,isok:= k.producers[key]
	if !isok{
		return nil
	}
	return p
}

func init() {
	sarama.Logger = &DebugLogger{}
	outputs.Add("kafka2", func() telegraf.Output {
		return &Kafka2{
			MaxRetry:     3,
			RequiredAcks: -1,
			producerFunc: sarama.NewSyncProducer,
		}
	})
}
