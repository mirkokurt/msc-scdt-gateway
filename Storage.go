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
	StorageQueue, err = dque.NewOrOpen(qName, qDir, segmentSize, ContactBuilder)
	if err != nil {
		log.Fatal("Error creating new dque ", err)
	}
}

func EnqueueContact(c StoredContact) {
	err := StorageQueue.Enqueue(&c)
	if err != nil {
		log.Warnf("Encode contact failed with: %v\n", err)
	}
}

func UploadContactsFromQueue() {

	ticker := time.NewTicker(600 * time.Second)
	for range ticker.C {
		for {
			var iface interface{}
			var err error

			// Dequeue the next item in the queue 
			if iface, err = StorageQueue.Dequeue(); err != nil  {
				break
			} else {
				log.Warnf(">>>>>>>>>>>>>>>>>>>>>>>>>> Dequeueing an object!\n")
				c, ok := iface.(*StoredContact)
				if !ok {
					log.Warnf("Dequeued object is not an StoredContact pointer")
				}
				
				// Reinsert the Contact into the store channel
				SplunkChannel <- *c

			}	
			time.Sleep(5 * time.Millisecond)
		}
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
