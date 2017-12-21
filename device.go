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

// NewDevice 创建一个新的日志输出设备
func NewDevice(define string) Device {
	var items = strings.SplitN(define, ":", 2)
	var name = items[0]
	var args string
	if len(items) == 2 {
		args = items[1]
	}
	return deviceMap[name](args)
}

// FileDevice 文件输出设备
type FileDevice struct {
	file     *os.File
	writer   *bufio.Writer
	prefix   string
	lock     sync.Mutex
	lastdate uint32
	rorate   int
}

func createFileDevice(args string) Device {
	return &FileDevice{
		prefix: args,
	}
}

func createFileHourDevice(args string) Device {
	return &FileDevice{
		prefix: args,
		rorate: HourRorate,
	}
}

func (file *FileDevice) Write(p []byte) {
	var err error
	var date uint32
	ldate := atomic.LoadUint32(&lastDate)
	if file.rorate == DayRorate {
		date = ldate
	} else if file.rorate == HourRorate {
		ltime := atomic.LoadUint32(&lastTime)
		date = ldate*100 + ltime/10000
	} else {
		date = ldate
	}
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
		filename := fmt.Sprintf("%s/logs/%s-%v.log", getCurrentParentDirectory(), file.prefix, date)
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

// Flush 刷新到设备
func (file *FileDevice) Flush() {
	file.lock.Lock()
	if file.writer != nil {
		file.writer.Flush()
	}
	file.lock.Unlock()
}

// ConsoleDevice 控制台设备
type ConsoleDevice struct {
	lock sync.Mutex
}

func createConsoleDevice(args string) Device {
	return &ConsoleDevice{}
}

func (console *ConsoleDevice) Write(p []byte) {
	console.lock.Lock()
	os.Stdout.Write(p)
	console.lock.Unlock()
}

// Flush 无行为
func (console *ConsoleDevice) Flush() {
}

// StdoutDevice 标准输出设备，定时刷新
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

// Write 写入
func (console *StdoutDevice) Write(p []byte) {
	console.lock.Lock()
	console.writer.Write(p)
	console.lock.Unlock()
}

// Flush 刷新
func (console *StdoutDevice) Flush() {
	console.lock.Lock()
	console.writer.Flush()
	console.lock.Unlock()
}

// NsqDevice nsq设备
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

// Flush 刷新
func (nsqd *NsqDevice) Flush() {
}

func substr(str string, start, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0
	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}
	return string(rs[start:end])
}

//获取父路径
func getParentDirectory(dirctory string) string {
	return substr(dirctory, 0, strings.LastIndex(dirctory, "/"))
}

//获取当前父路径
func getCurrentParentDirectory() string {
	return getParentDirectory(getCurrentDirectory())
}

//获取当前路径
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Printf("getCurrentDirectory fail: err=%v\n", err)
		return ""
	}
	return strings.Replace(dir, "\\", "/", -1)
}
