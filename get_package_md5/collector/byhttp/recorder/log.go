package recorder

import (
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"os"
	"sync"
)

type AccessRecorder struct {
	recordChannel   chan string
	errorRecordChan chan string
	wg              sync.WaitGroup
	errorFile       *os.File
	db              *leveldb.DB
}

func NewAccessRecorder(cache string) (*AccessRecorder, error) {
	db, err := leveldb.OpenFile(cache, nil)
	if err != nil {
		return nil, err
	}

	errorFile, err := os.OpenFile("error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	logger := &AccessRecorder{
		recordChannel:   make(chan string, 100),
		errorRecordChan: make(chan string, 100),
		errorFile:       errorFile,
		db:              db,
	}

	logger.wg.Add(2)
	go logger.processRecords()
	go logger.processErrorRecords()

	return logger, nil
}

func (l *AccessRecorder) Exist(url string) bool {
	if ok, err := l.db.Has([]byte(url), nil); ok {
		return true
	} else if err != nil {
		if err != leveldb.ErrNotFound {
			log.Println("Error searching from db:", err)
		}
		return false
	}
	return false
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
		if err := l.db.Put([]byte(record), []byte{0}, nil); err != nil {
			log.Println("Error writing to db:", err)
		}
	}
}

func (l *AccessRecorder) processErrorRecords() {
	defer l.wg.Done()
	for record := range l.errorRecordChan {
		if _, err := l.errorFile.WriteString(record + "\n"); err != nil {
			log.Println("Error writing to error file:", err)
		}
	}
}

func (l *AccessRecorder) Close() error {
	close(l.recordChannel)
	close(l.errorRecordChan)
	l.wg.Wait() // 等待所有记录都被处理完
	if err := l.db.Close(); err != nil {
		return err
	}
	return l.errorFile.Close()
}
