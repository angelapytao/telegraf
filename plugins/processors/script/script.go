package script

import (
	"fmt"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
	"github.com/influxdata/telegraf/plugins/processors/script/javascript"
)

type Script struct {
	Tag               string                 `toml:"tag_key"`                              // Processor ID for debug and metrics.
	Source            string                 `toml:"source"`                               // Inline script to execute.
	File              string                 `toml:"file"`                                 // Source file.
	Files             []string               `toml:"files"`                                // Multiple source files.
	Params            map[string]interface{} `toml:"params"`                               // Parameters to pass to script.
	Timeout           time.Duration          `toml:"timeout" validate:"min=0"`             // Execution timeout.
	TagOnException    string                 `toml:"tag_on_exception"`                     // Tag to add to events when an exception happens.
	MaxCachedSessions int                    `toml:"max_cached_sessions" validate:"min=0"` // Max. number of cached VM sessions.

	init      bool
	onChange  bool
	processor processors.Processor

	HttpInputFieldKey string `toml:"http_input_field_key"` // http input generate field name
	HttpInputTagKey   string `toml:"http_input_tag_key"`   // http input generate field name
}

func (s *Script) Apply(in ...telegraf.Metric) []telegraf.Metric {
	s.initOnce()
	for _, metric := range in {
		if field, ok := metric.GetField(s.HttpInputFieldKey); ok {
			tag, _ := metric.GetField(s.HttpInputTagKey)
			s.reLoadJavaScript(field.(string), tag.(string))
		}

		if s.processor != nil {
			s.processor.Run(metric)
		}
	}
	return in
}

// SampleConfig returns the default configuration of the Input
func (s *Script) SampleConfig() string {
	return ""
}

// Description returns a one-sentence description on the Input
func (s *Script) Description() string {
	return ""
}

func (s *Script) initOnce() {
	if s.init || (s.HttpInputFieldKey != "" && s.Source == "") {
		return
	}
	config := &javascript.Config{
		Tag:               s.Tag,
		Source:            s.Source,
		File:              s.File,
		Files:             s.Files,
		Params:            s.Params,
		Timeout:           s.Timeout,
		TagOnException:    s.TagOnException,
		MaxCachedSessions: 4,
	}
	p, err := javascript.New(config)
	if err != nil {
		fmt.Printf("init js vm err: %v", err)
	}

	s.processor = p
	s.init = true
}

func (s *Script) reLoadJavaScript(source, tag string) {
	config := &javascript.Config{
		Tag:               tag,
		Source:            source,
		File:              s.File,
		Files:             s.Files,
		Params:            s.Params,
		Timeout:           s.Timeout,
		TagOnException:    s.TagOnException,
		MaxCachedSessions: 4,
	}
	p, err := javascript.New(config)
	if err != nil {
		fmt.Printf("reInit js vm err: %v", err)
	}

	s.processor = p
}

func init() {
	processors.Add("script", func() telegraf.Processor {
		return &Script{}
	})
}
