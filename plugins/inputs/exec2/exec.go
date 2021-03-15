package exec2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	util "github.com/influxdata/telegraf/internal/config/frxs"
	common "github.com/influxdata/telegraf/plugins/common/http"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/outputs"
	"github.com/influxdata/telegraf/plugins/parsers"
	"github.com/influxdata/telegraf/plugins/parsers/nagios"
	"github.com/kballard/go-shellquote"
)

const sampleConfig = `
  ## Commands array
  commands = [
    "/tmp/test.sh",
    "/usr/bin/mycollector --foo=bar",
    "/tmp/collect_*.sh"
  ]

  ## pattern as argument for netstat find pid (ie, "netstat -anvp tcp|grep LISTEN|grep '\\<%s\\>' |awk '{print $9}'")
  pattern = "netstat -anvp tcp|grep LISTEN|grep '\\<%s\\>' |awk '{print $9}'"
  ## The listening port number of the process
  listen_ports ="80,8082"
  
  ## Timeout for each command to complete.
  timeout = "5s"

  ## When init gather ports err, the interval on retry.
  gather_err_interval = "120s"

  ## measurement name suffix (for separating different commands)
  name_suffix = "_mycollector"

  ## Data format to consume.
  ## Each data format has its own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_INPUT.md
  data_format = "influx"
`

const MaxStderrBytes = 512

type Exec2 struct {
	Commands []string
	Command  string

	Pattern  string
	Ports    string            `toml:"listen_ports"`
	cmd2Port map[string]string //<cmd, port>
	added    bool

	exCommands []string
	exCmd2Port map[string]string
	mutext     sync.RWMutex

	Timeout                internal.Duration
	GatherErrRetryInterval internal.Duration `toml:"gather_err_retry_interval"`
	parser                 parsers.Parser
	cancel                 context.CancelFunc
	// Debug is the option for running in debug mode
	Debug bool `toml:"debug"`

	runner Runner
	Log    telegraf.Logger `toml:"-"`

	// http client
	client *http.Client
	URL    string `toml:"url"`
	init   bool
}

type PortResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []int  `json:"data"`
}

func NewExec2() *Exec2 {
	return &Exec2{
		runner:  CommandRunner{},
		Timeout: internal.Duration{Duration: time.Second * 5},
	}
}

type Runner interface {
	Run(string, time.Duration) ([]byte, []byte, error)
}

type CommandRunner struct{}

func (c CommandRunner) Run(
	command string,
	timeout time.Duration,
) ([]byte, []byte, error) {
	split_cmd, err := shellquote.Split(command)
	if err != nil || len(split_cmd) == 0 {
		return nil, nil, fmt.Errorf("exec2: unable to parse command, %s", err)
	}

	cmd := exec.Command(split_cmd[0], split_cmd[1:]...)

	var (
		out    bytes.Buffer
		stderr bytes.Buffer
	)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	runErr := internal.RunTimeout(cmd, timeout)

	out = removeCarriageReturns(out)
	if stderr.Len() > 0 {
		stderr = removeCarriageReturns(stderr)
		stderr = truncate(stderr)
	}

	return out.Bytes(), stderr.Bytes(), runErr
}

func truncate(buf bytes.Buffer) bytes.Buffer {
	// Limit the number of bytes.
	didTruncate := false
	if buf.Len() > MaxStderrBytes {
		buf.Truncate(MaxStderrBytes)
		didTruncate = true
	}
	if i := bytes.IndexByte(buf.Bytes(), '\n'); i > 0 {
		// Only show truncation if the newline wasn't the last character.
		if i < buf.Len()-1 {
			didTruncate = true
		}
		buf.Truncate(i)
	}
	if didTruncate {
		buf.WriteString("...")
	}
	return buf
}

// removeCarriageReturns removes all carriage returns from the input if the
// OS is Windows. It does not return any errors.
func removeCarriageReturns(b bytes.Buffer) bytes.Buffer {
	if runtime.GOOS == "windows" {
		var buf bytes.Buffer
		for {
			byt, er := b.ReadBytes(0x0D)
			end := len(byt)
			if nil == er {
				end -= 1
			}
			if nil != byt {
				buf.Write(byt[:end])
			} else {
				break
			}
			if nil != er {
				break
			}
		}
		b = buf
	}
	return b

}

func (e *Exec2) ProcessCommand(command string, acc telegraf.Accumulator, wg *sync.WaitGroup) {
	defer wg.Done()
	_, isNagios := e.parser.(*nagios.NagiosParser)

	out, errbuf, runErr := e.runner.Run(command, e.Timeout.Duration)
	if !isNagios && runErr != nil {
		err := fmt.Errorf("exec2: %s for command '%s': %s", runErr, command, string(errbuf))
		acc.AddError(err)
		return
	}

	metrics, err := e.parser.Parse(out)
	if err != nil {
		acc.AddError(err)
		return
	}

	if isNagios {
		metrics, err = nagios.TryAddState(runErr, metrics)
		if err != nil {
			e.Log.Errorf("Failed to add nagios state: %s", err)
		}
	}

	for _, m := range metrics {
		e.addMetric(command, m, acc)
	}
}

func (e *Exec2) addMetric(command string, metric telegraf.Metric, acc telegraf.Accumulator) {
	// add port tag support
	if port, ok := e.cmd2Port[command]; ok {
		metric.AddTag("port", port)
	}

	e.addExMetric(command, metric)

	// filter "", set "0" as default.
	for k, v := range metric.Fields() {
		value := v.(string)
		if value == "" {
			metric.RemoveField(k)
			metric.AddField(k, "0")
		}

		mValue := strings.Split(value, "\n")
		if len(mValue) > 1 {
			e.Log.Infof("multi value: %v", mValue)
			metric.RemoveField(k)
			metric.AddField(k, mValue[0])
		}

	}

	acc.AddMetric(metric)
}

func (e *Exec2) addExMetric(command string, metric telegraf.Metric) {
	e.mutext.RLock()
	defer e.mutext.RUnlock()

	// add ex port tag support
	if port, ok := e.exCmd2Port[command]; ok {
		metric.AddTag("port", port)
	}
}

func (e *Exec2) SampleConfig() string {
	return sampleConfig
}

func (e *Exec2) Description() string {
	return "Read metrics from one or more commands that can output to stdout"
}

func (e *Exec2) SetParser(parser parsers.Parser) {
	e.parser = parser
}

func (e *Exec2) Gather(acc telegraf.Accumulator) error {
	var wg sync.WaitGroup

	// Legacy single command support
	if e.Command != "" {
		e.Commands = append(e.Commands, e.Command)
		e.Command = ""
	}

	commands := make([]string, 0, len(e.Commands))
	for _, pattern := range e.Commands {
		cmdAndArgs := strings.SplitN(pattern, " ", 2)
		if len(cmdAndArgs) == 0 {
			continue
		}

		matches, err := filepath.Glob(cmdAndArgs[0])
		if err != nil {
			acc.AddError(err)
			continue
		}

		if len(matches) == 0 {
			// There were no matches with the glob pattern, so let's assume
			// that the command is in PATH and just run it as it is
			commands = append(commands, pattern)
		} else {
			// There were matches, so we'll append each match together with
			// the arguments to the commands slice
			for _, match := range matches {
				if len(cmdAndArgs) == 1 {
					commands = append(commands, match)
				} else {
					commands = append(commands,
						strings.Join([]string{match, cmdAndArgs[1]}, " "))
				}
			}
		}
	}

	exCommands := e.readExCommandsLock(acc)

	wg.Add(len(commands) + len(exCommands))
	for _, command := range commands {
		go e.ProcessCommand(command, acc, &wg)
	}
	for _, command := range exCommands {
		go e.ProcessCommand(command, acc, &wg)
	}
	wg.Wait()
	return nil
}

// readCommandsLock mutext read commands
func (e *Exec2) readExCommandsLock(acc telegraf.Accumulator) []string {
	e.mutext.RLock()
	defer e.mutext.RUnlock()

	commands := make([]string, 0, len(e.exCommands))
	for _, pattern := range e.exCommands {
		cmdAndArgs := strings.SplitN(pattern, " ", 2)
		if len(cmdAndArgs) == 0 {
			continue
		}

		matches, err := filepath.Glob(cmdAndArgs[0])
		if err != nil {
			acc.AddError(err)
			continue
		}

		if len(matches) == 0 {
			// There were no matches with the glob pattern, so let's assume
			// that the command is in PATH and just run it as it is
			commands = append(commands, pattern)
		} else {
			// There were matches, so we'll append each match together with
			// the arguments to the commands slice
			for _, match := range matches {
				if len(cmdAndArgs) == 1 {
					commands = append(commands, match)
				} else {
					commands = append(commands,
						strings.Join([]string{match, cmdAndArgs[1]}, " "))
				}
			}
		}
	}
	return commands
}

// addPatternCommands split ports to generate multi commands by the specified pattern
func (e *Exec2) addPatternCommands() {
	if e.Pattern != "" && e.Ports != "" && !e.added {
		ports := strings.Split(e.Ports, ",")
		commands := make([]string, 0, len(ports))
		e.cmd2Port = make(map[string]string, len(ports))
		for _, port := range ports {
			cmd := fmt.Sprintf(e.Pattern, port)
			e.cmd2Port[cmd] = port
			commands = append(commands, cmd)
			e.added = true
		}
		e.Commands = append(e.Commands, commands...)
	}
}

// Connect satisfies the Ouput interface.
func (e *Exec2) Connect() error {
	return nil
}

// Close satisfies the Ouput interface.
func (e *Exec2) Close() error {
	return nil
}

func (e *Exec2) Start(acc telegraf.Accumulator) error {
	e.Log.Infof("Service start called...")
	return nil
}

func (e *Exec2) Stop() {
	e.Log.Info("Service stop called...")
	if e.cancel != nil {
		e.cancel()
	}
}

// Write writes the metrics to the configured command.
// receive metrics from http_listener_v2, add commands to the execute command list.
func (e *Exec2) Write(metrics []telegraf.Metric) error {
	e.Log.Infof("Received metrics: %v", metrics)

	ports := make([]string, 0)
	break_flag := false

	for _, m := range metrics {
		fields := m.FieldList()
		for _, f := range fields {
			// http_listener_v2 listen port had changed
			// notify insight console asyn
			if f.Key == "listen_port" {
				ip := m.Tags()["host"]
				port := f.Value.(string)
				e.Log.Infof("tag:%v, port:%v", ip, port)
				go e.notify(ip, port)
				go e.store(port)
			}

			if value, ok := f.Value.(float64); ok {
				if value == 0 {
					break_flag = true
					break
				}
				ports = append(ports, strconv.FormatFloat(value, 'f', -1, 64))
			}
		}

		if break_flag {
			break
		}
	}

	e.addExPatternCommands(ports)
	return nil
}

// notify insight console that http_listener_v2 listen port had changed
func (e *Exec2) notify(ip, port string) {
	body := fmt.Sprintf(`{
		"ip": "%s",
		"port": "%s"
	}`, ip, port)

	realUrl := e.URL + "/update"
	err := e.doHttpPost(realUrl, body)
	if err != nil {
		e.Log.Errorf("Notify Console Err: %v", err)

		var ctx context.Context
		ctx, e.cancel = context.WithCancel(context.Background())
		go e.gatherErrRetryInterval(realUrl, body, ctx, e.GatherErrRetryInterval.Duration)
	}
}

// store save the new port to local config
// for the next time recovery.
func (e *Exec2) store(port string) {
	path, _ := os.LookupEnv("CONFIG_DIR_D")
	e.Log.Infof("CONFIG_DIR_D: %s", path)
	if strings.TrimSpace(path) == "" {
		return
	}

	output := []byte(fmt.Sprintf(`[[inputs.http_listener_v2]]
	service_address = ":%s"
	data_format = "json"
	  `, port))

	e.Log.Infof("Marshaled:\n%s", output)

	err := ioutil.WriteFile(path+"http_listen_v2.conf", output, 0o644)
	if err != nil {
		e.Log.Errorf("WriteFile err: \n%v", err)
	}
}

func (e *Exec2) addExPatternCommands(ports []string) {
	if e.Pattern != "" {
		// write lock
		e.mutext.Lock()
		defer e.mutext.Unlock()

		// clear e.exCommands
		e.exCommands = make([]string, 0)

		commands := make([]string, 0, len(ports))
		e.exCmd2Port = make(map[string]string, len(ports))
		for _, port := range ports {
			cmd := fmt.Sprintf(e.Pattern, port)
			e.exCmd2Port[cmd] = port
			commands = append(commands, cmd)
		}

		e.exCommands = append(e.exCommands, commands...)
	}
}

func (e *Exec2) Init() error {
	// Legacy pattern command support
	e.addPatternCommands()

	// init ports list
	if !e.Debug && !e.init {
		err := e.httpInit()
		if err != nil {
			return err
		}

		localIP, err := util.GetAvaliableLocalIP()
		if err != nil {
			return err
		}

		realUrl := e.URL + "/getPorts?hostIp=" + localIP
		ports, err := e.gather(realUrl)
		if err != nil {
			e.Log.Errorf("Gather ports err: %v", err)

			var ctx context.Context
			ctx, e.cancel = context.WithCancel(context.Background())
			go e.gatherErrRetryInterval(realUrl, "", ctx, e.GatherErrRetryInterval.Duration)
		}
		e.addExPatternCommands(ports)

		e.init = true
	}

	return nil
}

func (e *Exec2) httpInit() error {
	e.client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
		Timeout: 6 * time.Second,
	}
	return nil
}

// gather get all ports from console by this ip as parameter.
// This is called by exec2 on initial
func (e *Exec2) gather(realUrl string) (ports []string, rspErr error) {
	acc := make(map[string]interface{}, 2)
	var wg sync.WaitGroup
	wg.Add(1)
	go func(url string, acc map[string]interface{}) {
		defer wg.Done()
		if err := e.gatherURL(realUrl, acc); err != nil {
			acc["err"] = fmt.Errorf("[url=%s]: %s", url, err)
		}
	}(realUrl, acc)
	wg.Wait()

	if v, ok := acc["ports"].([]string); ok {
		ports = append(ports, v...)
	}
	if v, ok := acc["err"].(error); ok {
		rspErr = v
	}
	return
}

func (e *Exec2) gatherErrRetryInterval(
	url string,
	body string,
	ctx context.Context,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		err := internal.SleepContext(ctx, interval)
		if err != nil {
			return
		}

		acc := make(map[string]interface{}, 1)

		err = e.gatherErrOnce(url, acc, body, interval)
		if err != nil {
			e.Log.Infof("gatherURL in err: %v", err)
		}

		select {
		case <-ctx.Done():
			e.Log.Infof("GatherOnErrRetry stoped.")
			return
		case <-ticker.C:
			e.Log.Infof("GatherOnErrRetry ticker in.")
			if v, ok := acc["ports"].([]string); ok {
				e.Log.Infof("Gather info: %v", v)
				e.addExPatternCommands(v)
				e.Log.Infof("GatherOnErrRetry return.")
				return
			}
			if body != "" && err == nil {
				e.Log.Infof("[%s] Notify console success.", url)
				return
			}
			e.Log.Infof("GatherOnErrRetry ticker out.")
		}
	}
}

func (e *Exec2) gatherErrOnce(
	url string,
	acc map[string]interface{},
	body string,
	timeout time.Duration) error {
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	done := make(chan error)
	go func() {
		if body == "" {
			done <- e.gatherURL(url, acc)
		} else {
			done <- e.doHttpPost(url, body)
		}
	}()

	for {
		select {
		case err := <-done:
			return err
		case <-ticker.C:
			e.Log.Infof("W! [exec2] [%s] did not complete within its interval", "gatherErrOnce")
		}
	}
}

func (e *Exec2) doHttpPost(url string, body string) error {
	if e.client == nil {
		err := e.httpInit()
		if err != nil {
			return err
		}
	}

	resp, err := common.HttpPost(e.client, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	e.Log.Infof("Response: %v", string(resp.Body))
	return nil
}

func (e *Exec2) gatherURL(url string, acc map[string]interface{}) (rspErr error) {
	resp, err := common.HttpGet(e.client, url)
	if err != nil {
		rspErr = err
		return
	}

	portResponse := &PortResponse{}
	err = json.Unmarshal(resp.Body, portResponse)
	if err != nil {
		rspErr = err
		return
	}

	e.Log.Infof("Response: %v", portResponse)

	ports := make([]string, len(portResponse.Data))
	for i, v := range portResponse.Data {
		ports[i] = strconv.Itoa(v)
	}

	acc["ports"] = ports
	return
}

func init() {
	exec := NewExec2()
	inputs.Add("exec2", func() telegraf.Input {
		return exec
	})
	outputs.Add("exec2", func() telegraf.Output {
		return exec
	})
}
