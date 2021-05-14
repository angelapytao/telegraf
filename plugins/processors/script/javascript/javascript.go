package javascript

import (
	"github.com/dop251/goja"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
	"github.com/pkg/errors"
)

type jsProcessor struct {
	*Config
	sessionPool *sessionPool
	sourceProg  *goja.Program
	sourceFile  string
	Tag         string
}

// New constructs a new Javascript processor from the given config
// object. It loads the sources, compiles them, and validates the entry point.
func New(c *Config) (processors.Processor, error) {
	err := c.Validate()
	if err != nil {
		return nil, err
	}

	sourceFile := "inline.js"
	sourceCode := []byte(c.Source)

	// Validate processor source code.
	prog, err := goja.Compile(sourceFile, string(sourceCode), true)
	if err != nil {
		return nil, err
	}

	pool, err := newSessionPool(prog, c)
	if err != nil {
		return nil, annotateError(c.Tag, err)
	}

	return &jsProcessor{
		Config:      c,
		sessionPool: pool,
		sourceProg:  prog,
		sourceFile:  sourceFile,
		Tag:         "JsProcessor",
	}, nil
}

func annotateError(id string, err error) error {
	if err == nil {
		return nil
	}
	if id != "" {
		return errors.Wrapf(err, "failed in processor.javascript with id=%v", id)
	}
	return errors.Wrap(err, "failed in processor.javascript")
}

// Run executes the processor on the given it event. It invokes the
// process function defined in the Javascript source.
func (p *jsProcessor) Run(event telegraf.Metric) (telegraf.Metric, error) {
	s := p.sessionPool.Get()
	defer p.sessionPool.Put(s)

	var rtn telegraf.Metric
	var err error

	rtn, err = s.runProcessFunc(event)
	return rtn, annotateError("p.Tag", err)
}

func (p *jsProcessor) String() string {
	return "script=[type=javascript, id=" + p.Tag + ", sources=" + p.sourceFile + "]"
}
