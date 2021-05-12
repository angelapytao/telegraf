module github.com/influxdata/telegraf

go 1.13

require (
	collectd.org v0.5.0
	github.com/Shopify/sarama v1.28.0
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15
	github.com/aws/aws-sdk-go v1.38.23
	github.com/dimchansky/utfbom v1.1.1
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-logfmt/logfmt v0.5.0
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/gobwas/glob v0.2.3
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/geo v0.0.0-20210211234256-740aa86cb551
	github.com/google/go-cmp v0.5.5
	github.com/influxdata/go-syslog/v2 v2.0.1
	github.com/influxdata/tail v1.0.1-0.20200707181643-03a791b270e4
	github.com/influxdata/toml v0.0.0-20190415235208-270119a8ce65
	github.com/influxdata/wlog v0.0.0-20160411224016-7c63b0a71ef8
	github.com/json-iterator/go v1.1.11
	github.com/kardianos/service v1.2.0
	github.com/karrick/godirwalk v1.16.1
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil v3.21.3+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.7.5
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/vjeantet/grok v1.0.1
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/sys v0.0.0-20210421221651-33663a62ff08
	golang.org/x/text v0.3.6
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7
)

// replaced due to https://github.com/satori/go.uuid/issues/73
replace github.com/satori/go.uuid => github.com/gofrs/uuid v3.2.0+incompatible
