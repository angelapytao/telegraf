// Copyright (c) 2015 HPE Software Inc. All rights reserved.
// Copyright (c) 2013 ActiveState Software Inc. All rights reserved.

package log

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/influxdata/telegraf/plugins/common/store"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/tail/ratelimiter"
	//"github.com/influxdata/tail/util"
	"github.com/influxdata/tail/watch"
	"gopkg.in/tomb.v1"
)

var (
	ErrStop = errors.New("tail should now stop")
)

type Line struct {
	Text   string
	Time   time.Time
	Err    error // Error from tail
	Offset int64
	FileDelCount int64
}

// NewLine returns a Line with present time.
func NewLine(text string) *Line {
	return &Line{text, time.Now(), nil, 0,0}
}

// SeekInfo represents arguments to `os.Seek`
type SeekInfo struct {
	Offset int64
	Whence int // os.SEEK_*
}

type logger interface {
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Fatalln(v ...interface{})
	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
	Panicln(v ...interface{})
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// Config is used to specify how a file must be tailed.
type Config struct {
	// File-specifc
	Location       *SeekInfo // Seek to this location before tailing
	ReOpen         bool      // Reopen recreated files (tail -F)
	MustExist      bool      // Fail early if the file does not exist
	Poll           bool      // Poll for file changes instead of using inotify
	Pipe           bool      // Is a named pipe (mkfifo)
	RateLimiter    *ratelimiter.LeakyBucket
	OpenReaderFunc func(rd io.Reader) io.Reader

	// Generic IO
	Follow      bool // Continue looking for new lines (tail -f)
	MaxLineSize int  // If non-zero, split longer lines into multiple lines

	// Logger, when nil, is set to tail.DefaultLogger
	// To disable logging: set field to tail.DiscardingLogger
	Logger logger
}

type Tail struct {
	Filename string
	Lines    chan *Line
	Config
	RegItems []RegexpItem
	file     *os.File
	reader   *bufio.Reader

	watcher watch.FileWatcher
	changes *watch.FileChanges

	tomb.Tomb // provides: Done, Kill, Dying

	lk sync.Mutex

	LastOffset int64

	FileDelCount int64 //日志文件删除或覆盖次数
}

var (
	// DefaultLogger is used when Config.Logger == nil
	DefaultLogger = log.New(os.Stderr, "", log.LstdFlags)
	// DiscardingLogger can be used to disable logging output
	DiscardingLogger = log.New(ioutil.Discard, "", 0)
)

// TailFile begins tailing the file. Output stream is made available
// via the `Tail.Lines` channel. To handle errors during tailing,
// invoke the `Wait` or `Err` method after finishing reading from the
// `Lines` channel.
func TailFile(filename string, config Config) (*Tail, error) {
	if config.ReOpen && !config.Follow {
		//util.Fatal("cannot set ReOpen without Follow.")
		Fatal("cannot set ReOpen without Follow.")
	}

	t := &Tail{
		Filename: filename,
		Lines:    make(chan *Line),
		Config:   config,
	}

	// when Logger was not specified in config, use default logger
	if t.Logger == nil {
		t.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	if t.Poll {
		t.watcher = watch.NewPollingFileWatcher(filename)
	} else {
		t.watcher = watch.NewInotifyFileWatcher(filename)
	}

	if t.MustExist {
		var err error
		t.file, err = OpenFile(t.Filename)
		if err != nil {
			return nil, err
		}
	}

	go t.tailFileSync()

	return t, nil
}

// Return the file's current position, like stdio's ftell().
// But this value is not very accurate.
// it may readed one line in the chan(tail.Lines),
// so it may lost one line.
func (tail *Tail) Tell() (offset int64, err error) {
	if tail.file == nil {
		return
	}
	offset, err = tail.file.Seek(0, os.SEEK_CUR)
	if err != nil {
		return
	}

	tail.lk.Lock()
	defer tail.lk.Unlock()
	if tail.reader == nil {
		return
	}
	offset -= int64(tail.reader.Buffered())
	return
}

// Stop stops the tailing activity.
func (tail *Tail) Stop() error {
	tail.Kill(nil)
	return tail.Wait()
}

// StopAtEOF stops tailing as soon as the end of the file is reached.
func (tail *Tail) StopAtEOF() error {
	tail.Kill(errStopAtEOF)
	return tail.Wait()
}

var errStopAtEOF = errors.New("tail: stop at eof")

func (tail *Tail) close() {
	close(tail.Lines)
	tail.closeFile()
}

func (tail *Tail) closeFile() {
	if tail.file != nil {
		tail.file.Close()
		tail.file = nil
	}
}

func (tail *Tail) reopen() error {
	tail.closeFile()
	for {
		var err error
		tail.file, err = OpenFile(tail.Filename)
		if err != nil {
			if os.IsNotExist(err) {
				tail.Logger.Printf("Waiting for %s to appear...", tail.Filename)
				if err := tail.watcher.BlockUntilExists(&tail.Tomb); err != nil {
					if err == tomb.ErrDying {
						return err
					}
					return fmt.Errorf("Failed to detect creation of %s: %s", tail.Filename, err)
				}
				continue
			}
			return fmt.Errorf("Unable to open file %s: %s", tail.Filename, err)
		}
		break
	}
	return nil
}

func (tail *Tail) readLine() (string, error) {
	tail.lk.Lock()
	line, err := tail.reader.ReadString('\n')
	tail.lk.Unlock()
	if err != nil {
		// Note ReadString "returns the data read before the error" in
		// case of an error, including EOF, so we return it as is. The
		// caller is expected to process it if err is EOF.
		return line, err
	}

	line = strings.TrimRight(line, "\n")

	return line, err
}

func (tail *Tail) tailFileSync() {
	defer tail.Done()
	defer tail.close()

	if !tail.MustExist {
		// deferred first open.
		err := tail.reopen()
		if err != nil {
			if err != tomb.ErrDying {
				tail.Kill(err)
			}
			return
		}
	}

	// Seek to requested location on first open of the file.
	if tail.Location != nil {
		_, err := tail.file.Seek(tail.Location.Offset, tail.Location.Whence)
		tail.Logger.Printf("Seeked %s - %+v\n", tail.Filename, tail.Location)
		if err != nil {
			tail.Killf("Seek error on %s: %s", tail.Filename, err)
			return
		}
	}

	tail.openReader()

	if err := tail.watchChanges(); err != nil {
		tail.Killf("Error watching for changes on %s: %s", tail.Filename, err)
		return
	}

	var offset int64 = -1
	var err error

	// Read line by line.
	for {
		// do not seek in named pipes
		if !tail.Pipe && offset > -1 {
			// grab the position in case we need to back up in the event of a half-line
			offset, err = tail.Tell()
			if err != nil {
				tail.Kill(err)
				return
			}
		}
		if offset == -1 {
			offset = tail.Location.Offset
		}

		line, err := tail.readLine()

		// Process `line` even if err is EOF.
		if err == nil {
			cooloff := !tail.sendLine(line, offset)
			if cooloff {
				// Wait a second before seeking till the end of
				// file when rate limit is reached.
				msg := ("Too much log activity; waiting a second " +
					"before resuming tailing")
				tail.Lines <- &Line{msg, time.Now(), errors.New(msg), offset,tail.FileDelCount}
				select {
				case <-time.After(time.Second):
				case <-tail.Dying():
					return
				}
				if err := tail.seekEnd(); err != nil {
					tail.Kill(err)
					return
				}
			}
		} else if err == io.EOF { //当文件指针读取到文件末尾
			tail.LastOffset = offset
			if !tail.Follow {
				if line != "" {
					tail.sendLine(line, offset)
				}
				return
			}
			if tail.Follow && line != "" {
				// this has the potential to never return the last line if
				// it's not followed by a newline; seems a fair trade here
				err := tail.seekTo(SeekInfo{Offset: offset, Whence: 0})
				if err != nil {
					tail.Kill(err)
					return
				}
			}
			// When EOF is reached, wait for more data to become
			// available. Wait strategy is based on the `tail.watcher`
			// implementation (inotify or polling).
			err := tail.waitForChanges()
			if err != nil {
				if err != ErrStop {
					tail.Kill(err)
				}
				return
			}
			//err = tail.seekTo(SeekInfo{Offset: offset, Whence: 0})
			err = tail.seekTo(SeekInfo{Offset: tail.LastOffset, Whence: 0})
			if err != nil {
				tail.Kill(err)
				return
			}
			offset=tail.LastOffset
		} else {
			// non-EOF error
			tail.Killf("Error reading %s: %s", tail.Filename, err)
			return
		}

		select {
		case <-tail.Dying():
			if tail.Err() == errStopAtEOF {
				continue
			}
			return
		default:
		}
	}
}

// watchChanges ensures the watcher is running.
func (tail *Tail) watchChanges() error {
	if tail.changes != nil {
		return nil
	}
	var pos int64
	var err error
	if !tail.Pipe {
		pos, err = tail.file.Seek(0, os.SEEK_CUR)
		if err != nil {
			return err
		}
	}
	tail.changes, err = tail.watcher.ChangeEvents(&tail.Tomb, pos)
	return err
}

// waitForChanges waits until the file has been appended, deleted,
// moved or truncated. When moved or deleted - the file will be
// reopened if ReOpen is true. Truncated files are always reopened.
func (tail *Tail) waitForChanges() error {
	if err := tail.watchChanges(); err != nil {
		return err
	}

	select {
	case <-tail.changes.Modified:
		return nil
	case <-tail.changes.Deleted:
		tail.changes = nil
		tail.FileDelCount++
		if tail.ReOpen {
			tail.LastOffset=0//文件删除后重新打开，将offset设置为0（文件开头）
			// XXX: we must not log from a library.
			tail.Logger.Printf("Re-opening moved/deleted file %s ...", tail.Filename)
			if err := tail.reopen(); err != nil {
				return err
			}
			if err:=store.SaveOffset(tail.Filename,0,tail.FileDelCount); err != nil {
				return err
			}
			tail.Logger.Printf("Successfully reopened %s", tail.Filename)
			tail.openReader()
			return nil
		} else {
			tail.Logger.Printf("Stopping tail as file no longer exists: %s", tail.Filename)
			return ErrStop
		}
	case <-tail.changes.Truncated:
		tail.LastOffset=0//文件Truncated后重新打开，将offset设置为0（文件开头）
		tail.FileDelCount++
		// Always reopen truncated files (Follow is true)
		tail.Logger.Printf("Re-opening truncated file %s ...", tail.Filename)
		if err := tail.reopen(); err != nil {
			return err
		}
		if err:=store.SaveOffset(tail.Filename,0,tail.FileDelCount); err != nil {
			return err
		}
		tail.Logger.Printf("Successfully reopened truncated %s", tail.Filename)
		tail.openReader()
		return nil
	case <-tail.Dying():
		return ErrStop
	}
	panic("unreachable")
}


func (tail *Tail) openReader() {
	tail.lk.Lock()
	var rd io.Reader = tail.file
	if tail.OpenReaderFunc != nil {
		rd = tail.OpenReaderFunc(rd)
	}

	if tail.MaxLineSize > 0 {
		// add 2 to account for newline characters
		tail.reader = bufio.NewReaderSize(rd, tail.MaxLineSize+2)
	} else {
		tail.reader = bufio.NewReader(rd)
	}
	tail.lk.Unlock()
}

func (tail *Tail) seekEnd() error {
	return tail.seekTo(SeekInfo{Offset: 0, Whence: os.SEEK_END})
}

func (tail *Tail) seekTo(pos SeekInfo) error {
	_, err := tail.file.Seek(pos.Offset, pos.Whence)
	if err != nil {
		return fmt.Errorf("Seek error on %s: %s", tail.Filename, err)
	}
	// Reset the read buffer whenever the file is re-seek'ed
	tail.reader.Reset(tail.file)
	return nil
}

// sendLine sends the line(s) to Lines channel, splitting longer lines
// if necessary. Return false if rate limit is reached.
func (tail *Tail) sendLine(line string, offset int64) bool {
	now := time.Now()
	lines := []string{line}

	// Split longer lines
	if tail.MaxLineSize > 0 && len(line) > tail.MaxLineSize {
		//lines = util.PartitionString(line, tail.MaxLineSize)
		lines = PartitionString(line, tail.MaxLineSize)
	}

	for _, line := range lines {
		tail.Lines <- &Line{line, now, nil, offset,tail.FileDelCount}
	}

	if tail.Config.RateLimiter != nil {
		ok := tail.Config.RateLimiter.Pour(uint16(len(lines)))
		if !ok {
			tail.Logger.Printf("Leaky bucket full (%v); entering 1s cooloff period.\n",
				tail.Filename)
			return false
		}
	}

	return true
}

// Cleanup removes inotify watches added by the tail package. This function is
// meant to be invoked from a process's exit handler. Linux kernel may not
// automatically remove inotify watches after the process exits.
func (tail *Tail) Cleanup() {
	watch.Cleanup(tail.Filename)
}

type Logger struct {
	*log.Logger
}

var LOGGER = &Logger{log.New(os.Stderr, "", log.LstdFlags)}

// fatal is like panic except it displays only the current goroutine's stack.
func Fatal(format string, v ...interface{}) {
	// https://github.com/hpcloud/log/blob/master/log.go#L45
	LOGGER.Output(2, fmt.Sprintf("FATAL -- "+format, v...)+"\n"+string(debug.Stack()))
	os.Exit(1)
}

// partitionString partitions the string into chunks of given size,
// with the last chunk of variable size.
func PartitionString(s string, chunkSize int) []string {
	if chunkSize <= 0 {
		panic("invalid chunkSize")
	}
	length := len(s)
	chunks := 1 + length/chunkSize
	start := 0
	end := chunkSize
	parts := make([]string, 0, chunks)
	for {
		if end > length {
			end = length
		}
		subLine := s[start:end]
		//在新截取的日志行前增加空格（多行匹配的正则表达式，这里暂时写死为空格），利用multiline的逻辑追加为多行类型的日志；否则将导致该行无法匹配正则表达式而丢失
		if start > 0 {
			subLine = " " + subLine
		}
		parts = append(parts, subLine)
		if end == length {
			break
		}
		start, end = end, end+chunkSize
	}
	return parts
}
