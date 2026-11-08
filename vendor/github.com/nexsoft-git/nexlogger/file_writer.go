package nexlogger

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var LogFile FileWriter
var defaultFilePath string = "./log"
var fileName = fmt.Sprintf("%s_%s.log", resource, application)
var fileMutex = sync.Mutex{}

func logFile() string {
	return fmt.Sprintf("%s/%s", defaultFilePath, fileName)
}

func SetFilePath(
	path string,
	filename string,
	extension string,
) {
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.Trim(path, "/")
	defaultFilePath = path
	fileName = filename
}

type FileWriter interface {
	Write([]byte) (int, error)
	getFile() *os.File
	setFile(*os.File)
	Close()
}

type fileWriter struct {
	file *os.File
}

func NewFileWriter() (
	result FileWriter,
	err error,
) {
	_ = os.MkdirAll(defaultFilePath, 0770)
	_ = os.MkdirAll(defaultFilePath+"/temp", 0770)

	file, err := os.OpenFile(
		logFile(),
		os.O_CREATE|os.O_APPEND|os.O_RDWR,
		0660,
	)

	if err != nil {
		return nil, err
	}

	return &fileWriter{
		file: file,
	}, nil
}

func (f *fileWriter) getFile() *os.File {
	return f.file
}

func (f *fileWriter) setFile(file *os.File) {
	f.file = file
}
func (f *fileWriter) Close() {
	if f.file != nil {
		_ = f.file.Close()
	}
}

func (f *fileWriter) Write(
	data []byte,
) (int, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	return f.file.WriteString(string(data))

}
