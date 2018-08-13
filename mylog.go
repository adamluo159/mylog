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
	_ LogLevel = iota
	LogDebug
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
	count        int
	pathfile     string
	locker       sync.Mutex
	fsize        ByteSize
	fmaxsize     ByteSize
	interval     time.Duration
	intervaltime time.Time
}

var gMyLog *MyLog = nil

func New(pathfile string, level LogLevel, interval time.Duration, fsize ByteSize) (*MyLog, error) {
	if pathfile == "" {
		return nil, fmt.Errorf("path empty")
	}

	if interval < time.Minute {
		return nil, fmt.Errorf("at least one minute interval time")
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

	if gMyLog == nil {
		gMyLog = l
	}

	intervalTime(l)
	return l, nil
}

func intervalTime(l *MyLog) {
	l.intervaltime = time.Now()
	if l.interval < time.Hour {
		l.intervaltime = l.intervaltime.Add(l.interval)
		return
	}
	t1 := l.intervaltime.Truncate(time.Hour)
	t2 := t1.Add(l.interval)
	t3 := t1.Add(time.Duration(24-t1.Hour()) * time.Hour)
	var ftime time.Duration
	if t3.After(t2) {
		ftime = t2.Sub(l.intervaltime)
	} else {
		ftime = t3.Sub(l.intervaltime)
	}
	l.intervaltime = l.intervaltime.Add(ftime)
}

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
	} else {
		file = filepath.Base(file)
	}
	t := time.Now()
	loghead := fmt.Sprintf("%s%s%s:%d  ", printLevel, t.Format("2006-01-02 15:04:05.000 "), file, line)
	logstr := fmt.Sprintf(loghead+format+"\n", a...)
	l.locker.Lock()
	if l.logfile == nil {
		l.locker.Unlock()
		return
	}
	n, err := l.logfile.WriteString(logstr)
	l.fsize += ByteSize(n)
	if err != nil {
		fmt.Println(err)
	}

	if t.After(l.intervaltime) {
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
	var filename string
	if next {
		if l.fsize == 0 {
			return
		}
		filename = fmt.Sprintf("%s%d%02d%02d_%02d",
			l.pathfile,
			l.intervaltime.Year(),
			l.intervaltime.Month(),
			l.intervaltime.Day(),
			l.intervaltime.Hour())
		if l.interval < time.Hour {
			filename = fmt.Sprintf("%s_%02d", filename, l.intervaltime.Minute())
		}
		l.count = 0
		l.intervaltime = l.intervaltime.Add(l.interval)
	} else {
		l.count++
		now := time.Now()
		filename = fmt.Sprintf("%s%d%02d%02d_%02d_%02d.%d",
			l.pathfile,
			now.Year(),
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
			l.count)
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
	l.doPrintf(LogWarn, printWarnLevel, format, a...)
}

func Debug(format string, a ...interface{}) {
	if gMyLog == nil {
		return
	}
	gMyLog.doPrintf(LogDebug, printDebugLevel, format, a...)
}

func Info(format string, a ...interface{}) {
	if gMyLog == nil {
		return
	}
	gMyLog.doPrintf(LogInfo, printInfoLevel, format, a...)
}

func Warn(format string, a ...interface{}) {
	if gMyLog == nil {
		return
	}
	gMyLog.doPrintf(LogWarn, printWarnLevel, format, a...)
}

func Error(format string, a ...interface{}) {
	if gMyLog == nil {
		return
	}
	gMyLog.doPrintf(LogError, printErrorLevel, format, a...)
}

func Fatal(format string, a ...interface{}) {
	if gMyLog == nil {
		return
	}
	gMyLog.doPrintf(LogFatal, printFatalLevel, format, a...)
}

func SetConsole(open bool) {
	if gMyLog != nil {
		gMyLog.console = open
	}
}

func Close() {
	gMyLog.Close()
}
