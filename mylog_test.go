package mylog

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestFunction(t *testing.T) {
	New("./log/testfunc.log", LogDebug, time.Hour, KB)
	Debug("%s", "debug")
	Info("%s", "info")
	Warn("%s", "warn")
	Error("%s", "error")
	Close()
}

func TestLogging(t *testing.T) {
	l, err := New("./log/logging.log", LogDebug, time.Minute, GB)
	if err != nil {
		fmt.Println(err)
		return
	}
	logstring := "testlogging"
	l.Debug(logstring)
	t1 := time.NewTimer(time.Minute * 2)
	t2 := time.NewTicker(time.Millisecond)
	count := 0
	for {
		select {
		case <-t1.C:
			fmt.Println("close")
			return
		case <-t2.C:
			count++
			l.Debug(logstring + strconv.Itoa(count))
			if count%60000 == 0 {
				fmt.Println("one minute")
			}
		}
	}
}

func TestGo(t *testing.T) {
	l, err := New("./log/goroute.log", LogDebug, time.Hour, GB)
	if err != nil {
		fmt.Println(err)
		return
	}
	var wglocker sync.WaitGroup
	wglocker.Add(2)
	go func() {
		count := 0
		logstring := "testgoroute1-"
		for {
			l.Debug(logstring + strconv.Itoa(count))
			count++
			time.Sleep(time.Millisecond * 1)
		}
		wglocker.Done()
	}()
	go func() {
		count := 0
		logstring := "testgoroute2-"
		for {
			l.Debug(logstring + strconv.Itoa(count))
			count++
			time.Sleep(time.Millisecond * 1)
		}
		wglocker.Done()
	}()
	wglocker.Wait()
}
