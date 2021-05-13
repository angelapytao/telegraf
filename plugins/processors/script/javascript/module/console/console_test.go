package console

// import (
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"go.uber.org/zap"

// 	"github.com/elastic/beats/v7/libbeat/beat"
// 	"github.com/elastic/beats/v7/libbeat/common"
// 	"github.com/elastic/beats/v7/libbeat/logp"
// 	"github.com/elastic/beats/v7/libbeat/processors/script/javascript"

// 	// Register require module.
// 	_ "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/require"
// )

// func TestConsole(t *testing.T) {
// 	const script = `
// var console = require('console');

// function process(evt) {
// 	console.debug("TestConsole Debug");
// 	console.log("TestConsole Log/Info");
// 	console.info("TestConsole Info %j", evt.fields);
// 	console.warn("TestConsole Warning [%s]", evt.fields.message);
// 	console.error("TestConsole Error processing event: %j", evt.fields);
// }
// `

// 	logp.DevelopmentSetup(logp.ToObserverOutput())
// 	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = p.Run(&beat.Event{Fields: common.MapStr{"message": "hello world!"}})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	logs := logp.ObserverLogs().FilterMessageSnippet("TestConsole").TakeAll()
// 	if assert.Len(t, logs, 5) {
// 		assert.Contains(t, logs[0].Message, "Debug")
// 		assert.Equal(t, logs[0].Level, zap.DebugLevel)

// 		assert.Contains(t, logs[1].Message, "Log/Info")
// 		assert.Equal(t, logs[1].Level, zap.InfoLevel)

// 		assert.Contains(t, logs[2].Message, "Info")
// 		assert.Equal(t, logs[2].Level, zap.InfoLevel)

// 		assert.Contains(t, logs[3].Message, "Warning")
// 		assert.Equal(t, logs[3].Level, zap.WarnLevel)

// 		assert.Contains(t, logs[4].Message, "Error")
// 		assert.Equal(t, logs[4].Level, zap.ErrorLevel)
// 	}
// }
