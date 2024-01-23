package recorder

import (
	"fmt"
	"os"
	"sync"
)

type AccessRecorder struct {
	recordChannel   chan string
	errorRecordChan chan string
	wg              sync.WaitGroup
	file            *os.File
	errorFile       *os.File
}

func NewAccessRecorder(filePath, errorFilePath string) (*AccessRecorder, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	errorFile, err := os.OpenFile(errorFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		file.Close()
		return nil, err
	}

	logger := &AccessRecorder{
		recordChannel:   make(chan string, 100),
		errorRecordChan: make(chan string, 100),
		file:            file,
		errorFile:       errorFile,
	}

	logger.wg.Add(2)
	go logger.processRecords()
	go logger.processErrorRecords()

	return logger, nil
}

func (l *AccessRecorder) Record(url string) {
	l.recordChannel <- url
}

func (l *AccessRecorder) RecordError(errorMessage string) {
	l.errorRecordChan <- errorMessage
}

func (l *AccessRecorder) processRecords() {
	defer l.wg.Done()
	for record := range l.recordChannel {
		if _, err := l.file.WriteString(record + "\n"); err != nil {
			fmt.Println("Error writing to file:", err)
		}
	}
}

func (l *AccessRecorder) processErrorRecords() {
	defer l.wg.Done()
	for record := range l.errorRecordChan {
		if _, err := l.errorFile.WriteString(record + "\n"); err != nil {
			fmt.Println("Error writing to error file:", err)
		}
	}
}

func (l *AccessRecorder) Close() error {
	close(l.recordChannel)
	close(l.errorRecordChan)
	l.wg.Wait() // 等待所有记录都被处理完
	if err := l.file.Close(); err != nil {
		return err
	}
	return l.errorFile.Close()
}
