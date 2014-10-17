package log

import (
	"bufio"
	"fmt"
	"github.com/bitly/go-nsq"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

func NewDevice(define string) Device {
	var items = strings.SplitN(define, ":", 2)
	var name = items[0]
	var args string
	if len(items) == 2 {
		args = items[1]
	}
	return deviceMap[name](args)
}

////////////////////////////////////////////////////////////////////////////////
// FileDevice
type FileDevice struct {
	file     *os.File
	writer   *bufio.Writer
	prefix   string
	lock     sync.Mutex
	lastdate uint32
}

func createFileDevice(args string) Device {
	return &FileDevice{
		prefix: args,
	}
}

func (file *FileDevice) Write(p []byte) {
	var err error
	date := atomic.LoadUint32(&lastDate)
	file.lock.Lock()
	if file.lastdate != date {
		if file.file != nil {
			file.writer.Flush()
			err = file.file.Close()
			if err != nil {
				fmt.Printf("ERROR: logger cannot close file: %v\n", err.Error())
			}
			file.file = nil
		}
	}
	if file.file == nil {
		filename := fmt.Sprintf("%s-%v.log", file.prefix, date)
		file.file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			file.lock.Unlock()
			fmt.Printf("ERROR: logger cannot open file: %v\n", err.Error())
			return
		}
		file.writer = bufio.NewWriter(file.file)
		file.lastdate = date
	}
	_, err = file.writer.Write(p)
	file.lock.Unlock()
	if err != nil {
		fmt.Printf("ERROR: logger cannot write file: %v\n", err.Error())
	}
	return
}

func (file *FileDevice) Flush() {
	file.lock.Lock()
	if file.writer != nil {
		file.writer.Flush()
	}
	file.lock.Unlock()
}

////////////////////////////////////////////////////////////////////////////////
// ConsoleDevice
type ConsoleDevice struct {
}

func createConsoleDevice(args string) Device {
	return &ConsoleDevice{}
}

func (console *ConsoleDevice) Write(p []byte) {
	os.Stdout.Write(p)
}

func (console *ConsoleDevice) Flush() {
}

////////////////////////////////////////////////////////////////////////////////
// StdoutDevice
type StdoutDevice struct {
	writer *bufio.Writer
	lock   sync.Mutex
}

func createStdoutDevice(args string) Device {
	var device = &StdoutDevice{
		writer: bufio.NewWriter(os.Stdout),
	}
	return device
}

func (console *StdoutDevice) Write(p []byte) {
	console.lock.Lock()
	console.writer.Write(p)
	console.lock.Unlock()
}

func (console *StdoutDevice) Flush() {
	console.lock.Lock()
	console.writer.Flush()
	console.lock.Unlock()
}

////////////////////////////////////////////////////////////////////////////////
// NsqDevice
type NsqDevice struct {
	writer *nsq.Producer
	name   string
	topic  string
}

func createNsqDevice(args string) Device {
	items := strings.SplitN(args, ":", 4)
	if len(items) != 4 {
		fmt.Printf("ERROR: logger init nsq, args invalid: %v\n", args)
		os.Exit(1)
	}
	for _, item := range items {
		if len(strings.Trim(item, " ")) == 0 {
			fmt.Printf("ERROR: logger init nsq, args invalid: %v\n", args)
			os.Exit(1)
		}
	}
	w, err := nsq.NewProducer(items[0]+":"+items[1], nsq.NewConfig())
	if err != nil {
		fmt.Printf("ERROR: logger init nsq: %v\n", err.Error())
		os.Exit(1)
	}
	return &NsqDevice{
		writer: w,
		name:   strings.Trim(items[1], " "),
		topic:  strings.Trim(items[2], " "),
	}
}

func (nsqd *NsqDevice) Write(p []byte) {
	var buff = buffs.get()
	buff.WriteString(nsqd.name)
	buff.WriteByte('|')
	buff.Write(p)
	var err = nsqd.writer.Publish(nsqd.topic, buff.Bytes())
	buffs.put(buff)
	if err != nil {
		fmt.Printf("ERROR: logger cannot write nsq: %v\n", err.Error())
	}
}

func (nsqd *NsqDevice) Flush() {
}
