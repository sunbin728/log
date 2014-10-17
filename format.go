package log

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync"
)

type buffPool struct {
	pool sync.Pool
}

var buffs = &buffPool{
	pool: sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0))
		},
	},
}

func (b *buffPool) get() *bytes.Buffer {
	return b.pool.Get().(*bytes.Buffer)
}

func (b *buffPool) put(buf *bytes.Buffer) {
	buf.Reset()
	b.pool.Put(buf)
}

type DefaultFormatter struct {
	format string
}

func getLevelStr(level int) byte {
	switch level {
	case DEBUG:
		return 'D'
	case INFO:
		return 'I'
	case WARN:
		return 'W'
	case ERROR:
		return 'E'
	case FATAL:
		return 'F'
	default:
		fmt.Printf("ERROR: logger level unknown: %v\n", level)
		return 'I'
	}
}

func (format *DefaultFormatter) Format(level int, msg string) []byte {
	buff := buffs.get()
	buff.WriteByte(getLevelStr(level))
	buff.WriteString(lastDateTimeStr)
	_, file, line, ok := runtime.Caller(2)
	if ok {
		buff.WriteByte(' ')
		var i int = len(file) - 2
		for ; i >= 0; i-- {
			if file[i] == '/' {
				i++
				break
			}
		}
		buff.WriteString(file[i:])
		buff.WriteByte(':')
		buff.WriteString(strconv.FormatInt(int64(line), 10))
	}
	buff.WriteString("] ")
	buff.WriteString(msg)
	buff.WriteByte('\n')
	var r = buff.Bytes()
	buffs.put(buff)
	return r
}
