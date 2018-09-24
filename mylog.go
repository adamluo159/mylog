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
	buf          []byte
}

func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

var gMyLog *MyLog = nil

func New(pathfile string, level LogLevel, interval time.Duration, fsize ByteSize) (*MyLog, error) {
	if pathfile == "" {
		return nil, fmt.Errorf("path empty")
	}

	if interval < time.Minute {
		interval = -1
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
	if l.interval < 0 {
		return
	}

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

func (l *MyLog) formatHeader(buf *[]byte, t *time.Time, file string, line int) {
	year, month, day := t.Date()
	itoa(buf, year, 4)
	*buf = append(*buf, '/')
	itoa(buf, int(month), 2)
	*buf = append(*buf, '/')
	itoa(buf, day, 2)
	*buf = append(*buf, ' ')
	hour, min, sec := t.Clock()
	itoa(buf, hour, 2)
	*buf = append(*buf, ':')
	itoa(buf, min, 2)
	*buf = append(*buf, ':')
	itoa(buf, sec, 2)
	*buf = append(*buf, '.')
	itoa(buf, t.Nanosecond()/1e3, 6)
	*buf = append(*buf, ' ')

	*buf = append(*buf, filepath.Base(file)...)
	*buf = append(*buf, ':')
	itoa(buf, line, -1)
	*buf = append(*buf, ": "...)
}
func (l *MyLog) doPrintf(level LogLevel, printLevel string, format string, a ...interface{}) {
	if level < l.level {
		return
	}
	s := fmt.Sprintf(printLevel+format, a...)

	now := time.Now()
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file)
	}

	l.locker.Lock()
	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, &now, file, line)
	l.buf = append(l.buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}

	if l.logfile == nil {
		l.locker.Unlock()
		return
	}

	n, err := l.logfile.Write(l.buf)
	l.fsize += ByteSize(n)
	if err != nil {
		fmt.Println(err)
	}

	if l.interval > time.Minute && now.After(l.intervaltime) {
		l.changeFile(true)
	} else if l.fsize >= l.fmaxsize {
		l.changeFile(false)
	}

	if l.console {
		fmt.Print(string(l.buf))
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

func (l *MyLog) Output(depth int, format string, a ...interface{}) {
	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file)
	}

	now := time.Now()
	l.locker.Lock()
	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, &now, file, line)
	l.buf = append(l.buf, format...)
	if len(format) == 0 || format[len(format)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}

	if l.logfile == nil {
		l.locker.Unlock()
		return
	}

	n, err := l.logfile.Write(l.buf)
	l.fsize += ByteSize(n)
	if err != nil {
		fmt.Println(err)
	}

	if l.interval > time.Minute && now.After(l.intervaltime) {
		l.changeFile(true)
	} else if l.fsize >= l.fmaxsize {
		l.changeFile(false)
	}

	l.locker.Unlock()

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
