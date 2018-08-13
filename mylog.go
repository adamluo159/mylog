package mylog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type LogLevel int

// levels
const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
	LogFatal
)

const (
	printDebugLevel = "[debug  ] "
	printInfoLevel  = "[info   ] "
	printErrorLevel = "[error  ] "
	printWarnLevel  = "[warn   ] "
	printFatalLevel = "[fatal  ] "
)

type ByteSize int64

const (
	_           = iota
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
	PB
)

type MyLog struct {
	level        LogLevel
	logfile      *os.File
	console      bool
	count        int8
	pathfile     string
	locker       sync.Mutex
	fsize        ByteSize
	fmaxsize     ByteSize
	interval     time.Duration
	intervaltime time.Time
}

func New(pathfile string, level LogLevel, interval time.Duration, fsize ByteSize) (*MyLog, error) {
	if pathfile == "" {
		return nil, fmt.Errorf("path empty")
	}
	err := os.MkdirAll(filepath.Dir(pathfile), os.ModePerm)
	if err != nil {
		return nil, err
	}

	l := &MyLog{
		level:    level,
		fmaxsize: fsize,
		interval: interval,
		pathfile: pathfile,
	}

	err = l.newFile()
	if err != nil {
		return nil, err
	}
	IntervalTime(l, interval)
	return l, nil
}

func IntervalTime(l *MyLog, i time.Duration) {
	now := time.Now()
	t1 := now.Truncate(time.Hour)
	t2 := t1.Add(i)
	t3 := t1.Add(time.Duration(24-t1.Hour()) * time.Hour)
	var ftime time.Duration
	if t3.After(t2) {
		ftime = t2.Sub(now)
	} else {
		ftime = t3.Sub(now)
	}
	time.AfterFunc(ftime, func() {
		l.locker.Lock()
		l.changeFile(true)
		l.locker.Unlock()
	})
}

// It's dangerous to call the method on logging
func (l *MyLog) Close() {
	l.locker.Lock()
	defer l.locker.Unlock()

	if l.logfile != nil {
		l.logfile.Close()
	}
	l.logfile = nil
}

func (l *MyLog) doPrintf(level LogLevel, printLevel string, format string, a ...interface{}) {
	if level < l.level {
		return
	}
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "???"
		line = 0
	}
	t := time.Now()
	loghead := fmt.Sprintf("%s%s%s:%d", t.Format("2006-01-02 15:04:05.000 "), printLevel, file, line)
	logstr := fmt.Sprintf(loghead+format+"\n", a...)
	l.locker.Lock()
	if l.logfile == nil {
		return
	}
	n, _ := l.logfile.WriteString(logstr)
	l.fsize += ByteSize(n)

	if t.Before(l.intervaltime) {
		l.changeFile(true)
	} else if l.fsize >= l.fmaxsize {
		l.changeFile(false)
	}
	if l.console {
		fmt.Print(logstr)
	}
	if level == LogFatal {
		os.Exit(1)
	}
	l.locker.Unlock()
}

func (l *MyLog) newFile() error {
	f, err := os.OpenFile(l.pathfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	stat, errStat := f.Stat()
	if errStat != nil {
		return errStat
	}

	l.logfile = f
	l.fsize = ByteSize(stat.Size())
	return nil
}

func (l *MyLog) changeFile(next bool) {
	l.logfile.Close()
	now := time.Now()
	filename := fmt.Sprintf("%s%d%02d%02d_%02d",
		l.pathfile,
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour())
	if next {
		l.count = 0
		l.intervaltime.Add(l.interval)
	} else {
		l.count++
		filename = fmt.Sprintf("%s.%d", filename, l.count)
	}

	os.Rename(l.pathfile, filename)
	l.newFile()
}

func (l *MyLog) Debug(format string, a ...interface{}) {
	l.doPrintf(LogDebug, printDebugLevel, format, a...)
}

func (l *MyLog) Info(format string, a ...interface{}) {
	l.doPrintf(LogInfo, printInfoLevel, format, a...)
}

func (l *MyLog) Error(format string, a ...interface{}) {
	l.doPrintf(LogError, printErrorLevel, format, a...)
}

func (l *MyLog) Fatal(format string, a ...interface{}) {
	l.doPrintf(LogFatal, printFatalLevel, format, a...)
}

func (l *MyLog) Warn(format string, a ...interface{}) {
	l.doPrintf(LogWarn, printInfoLevel, format, a...)
}

var gMyLog, _ = New("", LogDebug, 10*time.Minute, GB)

func Debug(format string, a ...interface{}) {
	gMyLog.doPrintf(LogDebug, printDebugLevel, format, a...)
}

func Info(format string, a ...interface{}) {
	gMyLog.doPrintf(LogInfo, printInfoLevel, format, a...)
}

func Warn(format string, a ...interface{}) {
	gMyLog.doPrintf(LogWarn, printInfoLevel, format, a...)
}

func Error(format string, a ...interface{}) {
	gMyLog.doPrintf(LogError, printErrorLevel, format, a...)
}

func Fatal(format string, a ...interface{}) {
	gMyLog.doPrintf(LogFatal, printFatalLevel, format, a...)
}

func Close() {
	gMyLog.Close()
}
