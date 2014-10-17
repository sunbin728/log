package log

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

const (
	DEBUG   = iota
	INFO    = iota
	WARN    = iota
	ERROR   = iota
	DISABLE = iota
	FATAL   = iota
)

type Logger struct {
	maxLevel int
	format   Formatter
	writers  []Writer
}

type Writer struct {
	level  int
	device Device
}

type Device interface {
	Write(msg []byte)
	Flush()
}

type Formatter interface {
	Format(level int, msg string) []byte
}

type LoggerDefine struct {
	Name   string `toml:"name"`
	Level  string `toml:"level"`
	Writer string `toml:"writer"`
}

type LoggerConfig struct {
	Logger []LoggerDefine `toml:"logger"`
}

var (
	lastDateTimeStr string
	lastTime        uint32
	lastDate        uint32
	deviceMap       = map[string]func(string) Device{
		"file":    createFileDevice,
		"console": createConsoleDevice,
		"nsq":     createNsqDevice,
	}
	defaultLogger      = NewLogger(&DefaultFormatter{}, NewWriter(DEBUG, "console"))
	loggerMap          = map[string]*Logger{}
	ErrNameNotFound    = errors.New("name_not_found")
	ErrIndexOutOfBound = errors.New("index_out_of_bound")
)

func Init(config []LoggerDefine) {
	for _, logger := range config {
		logger.Name = strings.ToLower(logger.Name)
		logger.Writer = strings.ToLower(logger.Writer)
		var log, ok = loggerMap[logger.Name]
		if !ok {
			log = NewLogger(&DefaultFormatter{}, NewWriter(getLevelFromStr(logger.Level), logger.Writer))
		} else {
			log.writers = append(log.writers, NewWriter(getLevelFromStr(logger.Level), logger.Writer))
			log.UpdateLevel()
		}
		loggerMap[logger.Name] = log
		if logger.Name == "default" {
			defaultLogger = log
		}
	}
	updateNow()
	go func() {
		for {
			time.Sleep(time.Second)
			updateNow()
			defaultLogger.Flush()
			for _, log := range loggerMap {
				log.Flush()
			}
		}
	}()
}

func InitFromStr(tomlstr string) {
	var config LoggerConfig

	var _, err = toml.Decode(tomlstr, &config)
	if err != nil {
		fmt.Printf("ERROR: logger read config: %v\n", err.Error())
		os.Exit(1)
	}
	Init(config.Logger)
}

func InitFromFile(configFile string) {
	var tomlstr, err = ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("ERROR: logger read config: %v\n", err.Error())
		os.Exit(1)
	}
	InitFromStr(string(tomlstr))
}

func GetLogger(name string) *Logger {
	var logger, ok = loggerMap[name]
	if ok {
		return logger
	} else {
		return defaultLogger
	}
}

func getLevelFromStr(level string) int {
	switch strings.ToLower(level) {
	case "d":
		return DEBUG
	case "i":
		return INFO
	case "w":
		return WARN
	case "e":
		return ERROR

	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "warning":
		return WARN
	case "err":
		return ERROR
	case "error":
		return ERROR
	case "disable":
		return DISABLE
	default:
		fmt.Printf("ERROR: logger level unknown: %v\n", level)
		return INFO
	}
}

func SetLevel(name string, index int, level string) error {
	var log *Logger
	if name == "default" {
		log = defaultLogger
	}
	if l, ok := loggerMap[name]; !ok {
		fmt.Printf("ERROR: log name not found: %v\n", name)
		return ErrNameNotFound
	} else {
		log = l
	}
	if index >= len(log.writers) {
		fmt.Printf("ERROR: log index exceed: %v, %v\n", len(log.writers), index)
		return ErrIndexOutOfBound
	}
	var newlevel = getLevelFromStr(level)
	if index == -1 {
		for _, writer := range log.writers {
			writer.level = newlevel
		}
	} else {
		log.writers[index].level = newlevel
	}
	log.UpdateLevel()
	return nil
}

func NewLogger(format Formatter, writers ...Writer) *Logger {
	var logger = Logger{
		format:  format,
		writers: writers,
	}
	logger.UpdateLevel()
	return &logger
}

func NewWriter(level int, device string) Writer {
	return Writer{
		level:  level,
		device: NewDevice(device),
	}
}

func (log *Logger) UpdateLevel() {
	log.maxLevel = DISABLE
	for _, writer := range log.writers {
		if writer.level < log.maxLevel {
			log.maxLevel = writer.level
		}
	}
}

func updateNow() {
	t := time.Now()
	dt := uint32(t.Year()%100*10000 + int(t.Month())*100 + t.Day())
	tm := uint32(t.Hour()*10000 + t.Minute()*100 + t.Second())
	atomic.StoreUint32(&lastDate, dt)
	atomic.StoreUint32(&lastTime, tm)
	lastDateTimeStr = fmt.Sprintf("%04d %06d", dt%10000, tm)
}

func (log *Logger) Flush() {
	for _, writer := range log.writers {
		writer.device.Flush()
	}
}

func (log *Logger) Write(level int, format string, a ...interface{}) {
	if log.maxLevel > level {
		return
	}
	var msg string
	if len(a) == 0 {
		msg = format
	} else {
		msg = fmt.Sprintf(format, a...)
	}
	var buff = log.format.Format(level, msg)
	for _, writer := range log.writers {
		if level >= writer.level {
			writer.device.Write(buff)
		}
	}
}

func Debug(format string, a ...interface{}) {
	defaultLogger.Write(DEBUG, format, a...)
}

func Info(format string, a ...interface{}) {
	defaultLogger.Write(INFO, format, a...)
}

func Warn(format string, a ...interface{}) {
	defaultLogger.Write(WARN, format, a...)
}

func Error(format string, a ...interface{}) {
	defaultLogger.Write(ERROR, format, a...)
}

func Fatal(format string, a ...interface{}) {
	defaultLogger.Write(FATAL, format, a...)
	os.Exit(1)
}

func (logger *Logger) Debug(format string, a ...interface{}) {
	logger.Write(DEBUG, format, a...)
}

func (logger *Logger) Info(format string, a ...interface{}) {
	logger.Write(INFO, format, a...)
}

func (logger *Logger) Warn(format string, a ...interface{}) {
	logger.Write(WARN, format, a...)
}

func (logger *Logger) Error(format string, a ...interface{}) {
	logger.Write(ERROR, format, a...)
}

func (logger *Logger) Fatal(format string, a ...interface{}) {
	logger.Write(FATAL, format, a...)
	os.Exit(1)
}

func main() {
	Debug("hello")
	time.Sleep(time.Hour)
}
