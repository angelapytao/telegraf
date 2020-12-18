module github.com/influxdata/telegraf

go 1.12

require (
	collectd.org v0.3.0
	github.com/Shopify/sarama v1.24.1
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6
	github.com/alecthomas/units v0.0.0-20190717042225-c3de453c63f4
	github.com/aws/aws-sdk-go v1.19.41
	github.com/go-logfmt/logfmt v0.4.0
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/gobwas/glob v0.2.3
	github.com/gofrs/uuid v2.1.0+incompatible
	github.com/golang/geo v0.0.0-20190916061304-5b978397cfec
	github.com/google/go-cmp v0.4.0
	github.com/influxdata/go-syslog/v2 v2.0.1
	github.com/influxdata/toml v0.0.0-20190415235208-270119a8ce65
	github.com/influxdata/wlog v0.0.0-20160411224016-7c63b0a71ef8
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/kardianos/service v1.0.0
	github.com/karrick/godirwalk v1.12.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/klauspost/compress v1.9.2 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/shirou/gopsutil v2.20.2+incompatible
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	github.com/tidwall/gjson v1.3.0
	github.com/vjeantet/grok v1.0.0
	golang.org/x/crypto v0.0.0-20200204104054-c9f3fb736b72 // indirect
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a // indirect
	golang.org/x/sys v0.0.0-20200615200032-f1bc736245b1
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.3.0 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

// replaced due to https://github.com/satori/go.uuid/issues/73
replace github.com/satori/go.uuid => github.com/gofrs/uuid v3.2.0+incompatible
