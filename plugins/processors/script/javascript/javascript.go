package javascript

import (
	"runtime"
	"strings"

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
	// stats       *processorStats
	Tag string
}

// New constructs a new Javascript processor from the given config
// object. It loads the sources, compiles them, and validates the entry point.
func New(c *Config) (processors.Processor, error) {
	err := c.Validate()
	if err != nil {
		return nil, err
	}

	var sourceFile string
	var sourceCode []byte

	switch {
	case c.Source != "":
		sourceFile = "inline.js"
		sourceCode = []byte(c.Source)
		// case c.File != "":
		// 	sourceFile, sourceCode, err = loadSources(c.File)
		// case len(c.Files) > 0:
		// 	sourceFile, sourceCode, err = loadSources(c.Files...)
	}
	// if err != nil {
	// 	return nil, annotateError(c.Tag, err)
	// }

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
		Tag:         "TestJsProcessorTag",
		// stats:       getStats(c.Tag, reg),
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
	// if p.stats == nil {
	// 	rtn, err = s.runProcessFunc(event)
	// } else {
	// 	rtn, err = p.runWithStats(s, event)
	// }
	return rtn, annotateError("p.Tag", err)
}

// func (p *jsProcessor) runWithStats(s *session, event telegraf.Metric) (telegraf.Metric, error) {
// 	start := time.Now()
// 	event, err := s.runProcessFunc(event)
// 	elapsed := time.Since(start)

// 	p.stats.processTime.Update(int64(elapsed))
// 	if err != nil {
// 		p.stats.exceptions.Inc()
// 	}
// 	return event, err
// }

func (p *jsProcessor) String() string {
	return "script=[type=javascript, id=" + p.Tag + ", sources=" + p.sourceFile + "]"
}

// hasMeta reports whether path contains any of the magic characters
// recognized by Match/Glob.
func hasMeta(path string) bool {
	magicChars := `*?[`
	if runtime.GOOS != "windows" {
		magicChars = `*?[\`
	}
	return strings.ContainsAny(path, magicChars)
}
