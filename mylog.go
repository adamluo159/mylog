package mylog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// levels
const (
	debugLevel = 0
	infoLevel  = 1
	errorLevel = 2
	warnLevel  = 3
	fatalLevel = 4
)

const (
	printDebugLevel = "[debug  ] "
	printInfoLevel  = "[info   ] "
	printErrorLevel = "[error  ] "
	printWarnLevel  = "[warn   ] "
	printFatalLevel = "[fatal  ] "
)

var (
	myLogs       []*MyLogger
	logsrwlocker sync.RWMutex
)

type MyLogger struct {
	level        int
	baseMyLogger *log.Logger
	baseFile     *os.File
	pathString   string
	printConsole bool
	count        int8
	filesize     uint64
	locker       sync.Mutex
	flag         int
}

func loopTime() {
	h := 25 * time.Hour
	now := time.Now()
	t1 := now.Truncate(time.Hour)
	t2 := t1.Add(h)
	t3 := t1.Add(time.Duration(24-t1.Hour()) * time.Hour)

	var sleepDuration time.Duration
	if t3.After(t2) {
		sleepDuration = t2.Sub(now)
	} else {
		sleepDuration = t3.Sub(now)
	}
	time.Sleep(sleepDuration)
	for {
		for i := 0; i < len(myLogs); i++ {
			myLogs[i].changeFile(true)
		}
		time.Sleep(h)
	}
}

func New(strLevel string, pathFile string, flag int) (*MyLogger, error) {
	if pathFile == "" {
		return nil, fmt.Errorf("path empty")
	}

	for i := 0; i < len(myLogs); i++ {
		if myLogs[i].pathString == pathFile {
			return nil, fmt.Errorf("file already open path:%s", pathFile)
		}
	}

	// level
	var level int
	switch strings.ToLower(strLevel) {
	case "debug":
		level = debugLevel
	case "info":
		level = infoLevel
	case "error":
		level = errorLevel
	case "warn":
		level = warnLevel
	case "fatal":
		level = fatalLevel
	default:
		return nil, fmt.Errorf("unknown level: %s", strLevel)
	}

	err := os.MkdirAll(filepath.Dir(pathFile), os.ModePerm)
	if err != nil {
		return nil, err
	}

	logger := &MyLogger{
		level:      level,
		pathString: pathFile,
	}

	err = logger.newFile()
	if err != nil {
		return nil, err
	}

	myLogs = append(myLogs, logger)

	return logger, nil
}

// It's dangerous to call the method on logging
func (logger *MyLogger) Close() {
	if logger.baseFile != nil {
		logger.baseFile.Close()
	}

	logger.baseMyLogger = nil
	logger.baseFile = nil
}

func (logger *MyLogger) doPrintf(level int, printLevel string, format string, a ...interface{}) {
	if level < logger.level {
		return
	}
	if logger.baseMyLogger == nil {
		panic("logger closed")
	}
	logstr := fmt.Sprintf(printLevel+format+"\n", a...)
	logger.baseMyLogger.Output(3, format)
	if logger.printConsole {
		fmt.Print(logstr)
	}
	if level == fatalLevel {
		os.Exit(1)
	}
}

func (logger *MyLogger) newFile() error {
	f, err := os.OpenFile(filepath.Base(logger.pathString), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	logger.baseFile = f
	logger.filesize = 0
	logger.baseMyLogger = log.New(f, "", logger.flag)
	return nil
}

func (logger *MyLogger) changeFile(next bool) {
	logger.locker.Lock()
	defer logger.locker.Unlock()

	now := time.Now()
	filename := fmt.Sprintf("%s%d%02d%02d_%02d",
		logger.pathString,
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour())
	if next {
		logger.count = 0
	} else {
		logger.count++
		filename = fmt.Sprintf("%s.%d", filename, logger.count)
	}

	logger.baseFile.Close()
	os.Rename(logger.pathString, filename)

	logger.newFile()
}

func (logger *MyLogger) Debug(format string, a ...interface{}) {
	logger.doPrintf(debugLevel, printDebugLevel, format, a...)
}

func (logger *MyLogger) Info(format string, a ...interface{}) {
	logger.doPrintf(infoLevel, printInfoLevel, format, a...)
}

func (logger *MyLogger) Error(format string, a ...interface{}) {
	logger.doPrintf(errorLevel, printErrorLevel, format, a...)
}

func (logger *MyLogger) Fatal(format string, a ...interface{}) {
	logger.doPrintf(fatalLevel, printFatalLevel, format, a...)
}

func (logger *MyLogger) Warn(format string, a ...interface{}) {
	logger.doPrintf(warnLevel, printInfoLevel, format, a...)
}

var gMyLogger, _ = New("debug", "", log.LstdFlags)

func Debug(format string, a ...interface{}) {
	gMyLogger.doPrintf(debugLevel, printDebugLevel, format, a...)
}

func Info(format string, a ...interface{}) {
	gMyLogger.doPrintf(infoLevel, printInfoLevel, format, a...)
}

func Warn(format string, a ...interface{}) {
	gMyLogger.doPrintf(warnLevel, printInfoLevel, format, a...)
}

func Error(format string, a ...interface{}) {
	gMyLogger.doPrintf(errorLevel, printErrorLevel, format, a...)
}

func Fatal(format string, a ...interface{}) {
	gMyLogger.doPrintf(fatalLevel, printFatalLevel, format, a...)
}

func Close() {
	gMyLogger.Close()
}
