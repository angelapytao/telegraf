package path_test

// import (
// 	"testing"

// 	"github.com/stretchr/testify/assert"

// 	"github.com/elastic/beats/v7/libbeat/beat"
// 	"github.com/elastic/beats/v7/libbeat/common"
// 	"github.com/elastic/beats/v7/libbeat/processors/script/javascript"

// 	_ "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/path"
// 	_ "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/require"
// )

// func TestWin32(t *testing.T) {
// 	const script = `
// var path = require('path');

// function process(evt) {
//     var filename = "C:\\Windows\\system32\\..\\system32\\system32.dll";
// 	evt.Put("result", {
//         raw: filename,
//     	basename: path.win32.basename(filename),
//     	dirname:  path.win32.dirname(filename),
//     	extname: path.win32.extname(filename),
//     	isAbsolute: path.win32.isAbsolute(filename),
//     	normalize: path.win32.normalize(filename),
//         sep: path.win32.sep,
//     });
// }
// `

// 	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	evt, err := p.Run(&beat.Event{Fields: common.MapStr{}})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	fields := evt.Fields.Flatten()
// 	assert.Equal(t, "system32.dll", fields["result.basename"])
// 	assert.Equal(t, `C:\Windows\system32`, fields["result.dirname"])
// 	assert.Equal(t, ".dll", fields["result.extname"])
// 	assert.Equal(t, true, fields["result.isAbsolute"])
// 	assert.Equal(t, `C:\Windows\system32\system32.dll`, fields["result.normalize"])
// 	assert.EqualValues(t, '\\', fields["result.sep"])
// }

// func TestPosix(t *testing.T) {
// 	const script = `
// var path = require('path');

// function process(evt) {
//     var filename = "/usr/lib/../lib/libcurl.so";
// 	evt.Put("result", {
//         raw: filename,
//     	basename: path.posix.basename(filename),
//     	dirname:  path.posix.dirname(filename),
//     	extname: path.posix.extname(filename),
//     	isAbsolute: path.posix.isAbsolute(filename),
//     	normalize: path.posix.normalize(filename),
//         sep: path.posix.sep,
//     });
// }
// `

// 	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	evt, err := p.Run(&beat.Event{Fields: common.MapStr{}})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	fields := evt.Fields.Flatten()
// 	assert.Equal(t, "libcurl.so", fields["result.basename"])
// 	assert.Equal(t, "/usr/lib", fields["result.dirname"])
// 	assert.Equal(t, ".so", fields["result.extname"])
// 	assert.Equal(t, true, fields["result.isAbsolute"])
// 	assert.Equal(t, "/usr/lib/libcurl.so", fields["result.normalize"])
// 	assert.EqualValues(t, '/', fields["result.sep"])
// }
