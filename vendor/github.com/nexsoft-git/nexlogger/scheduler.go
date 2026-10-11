package nexlogger

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/robfig/cron"
)

var s1 = rand.NewSource(time.Now().UnixNano())
var r1 = rand.New(s1)

var cronScheduler string = "0 0 0 * * *"

func SetCronSchedulerWrapLog(c string) {
	cronScheduler = c
}

func startSchedulingWrapLogFile() {
	if LogFile != nil {
		scheduler := cron.New()
		scheduler.AddFunc(cronScheduler, wrapLogFile)
		scheduler.Start()
	}
}

func wrapLogFile() {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	_ = LogFile.getFile().Close()
	fileTemp := fmt.Sprintf("%s/%s", defaultFilePath+"/temp", fileName)

	_ = os.Rename(logFile(), fileTemp)

	zipFileName := fmt.Sprintf("%s_%s_log_%s_%d.zip", resource, application, time.Now().Format("2006_02_01"), r1.Intn(9999))

	_ = zipFiles(defaultFilePath+"/"+zipFileName, []string{fileTemp})
	_ = os.Remove(fileTemp)
	_ = os.Remove(logFile())

	file, _ := os.OpenFile(
		logFile(),
		os.O_CREATE|os.O_APPEND|os.O_RDWR,
		0660,
	)

	LogFile.setFile(file)

}
