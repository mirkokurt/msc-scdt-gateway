package main

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/joncrlsn/dque"
)

var StorageQueue *dque.DQue

func ContactBuilder() interface{} {
	return &StoredContact{}
}

func OpenQueue() {
	qName := "StorageContactQueue"
	qDir := "."
	segmentSize := 500

	var err error
	StorageQueue, err = dque.Open(qName, qDir, segmentSize, ContactBuilder)
	if err != nil {
		StorageQueue, err = dque.New(qName, qDir, segmentSize, ContactBuilder)
	}
	check(err)
}

func EnqueueContact(c StoredContact) {
	err := StorageQueue.Enqueue(&c)
	log.Warn("Encode contact failed with: %v\n", err)
}

func UploadContactsFromQueue() {
	for {

		var iface interface{}
		var err error

		// Dequeue the next item in the queue and block until one is available
		if iface, err = StorageQueue.DequeueBlock(); err != nil {
			log.Warn("Error dequeuing item ", err)
		}
		c, ok := iface.(*StoredContact)
		if !ok {
			log.Warn("Dequeued object is not an StoredContact pointer")
		}

		// Reinsert the Contact into the store channel
		SplunkChannel <- c

		time.Sleep(5 * time.Second)
	}
	if StorageQueue != nil {
		StorageQueue.Close()
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
