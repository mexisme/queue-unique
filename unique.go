package queue

import (
	"sync"

	log "gopkg.in/Sirupsen/logrus.v0"
)

type UniqueQueue struct {
	MatcherID   func(interface{}) string // Callback to get an ID that can be used to identify whether an incoming item is already in the queue
	In          <-chan interface{}       // Incoming Item queue
	Out         chan<- interface{}       // Queue for workers
	QueueLength int
	uniqueIDs   map[string]bool  // Set of URLs, ordered by incoming
	feeder      chan interface{} // Internal queue for determining when to pop an item off the `uniqueIDs` set
	feederOk    bool
	wg          sync.WaitGroup
}

func (q *UniqueQueue) Init() *UniqueQueue {
	if q.QueueLength == 0 {
		q.QueueLength = 100
	}

	q.feederOk = true
	q.uniqueIDs = make(map[string]bool)
	// The queue length should probably be configurable:
	q.feeder = make(chan interface{}, q.QueueLength)

	return q
}

func (q *UniqueQueue) Run() {
	if cap(q.Out) == 0 {
		panic("Outgoing queue needs to be > 0 to avoid deadlocks")
	}
	q.wg.Add(1)
	go q.fifo()
}

func (q *UniqueQueue) Close() {
	q.feederOk = false
	q.wg.Wait()
}

// Push the data from one Q to the next, uniquifying on the URL
func (q *UniqueQueue) fifo() {
	// If the In is closed, there's nothing incoming:
	defer close(q.feeder)
	defer q.wg.Done()

	inQueueOk := true
	counter := 0

	for q.feederOk && (inQueueOk || len(q.feeder) > 0) {
		select {
		case newItem, inQueueOk := <-q.In:
			if inQueueOk {
				counter++
				log.WithFields(log.Fields{
					"Count": counter,
				}).Debug("Incoming item")
				q.pushUnique(newItem)
			}

		default:
			// We don't want this to block the rest of the loop
		}

		// If the Out has room:
		if len(q.Out) < cap(q.Out) {
			q.popTo(func(newItem interface{}) {
				q.Out <- newItem
			})
		}
	}
}

func (q *UniqueQueue) getID(item interface{}) string {
	// I'd rather give meaningful failures
	if q.MatcherID == nil {
		panic("MatcherID func() not provided")
	}

	return q.MatcherID(item)
}

func (q *UniqueQueue) pushUnique(item interface{}) bool {
	id := q.getID(item)

	log.WithFields(log.Fields{
		"ID":   id,
		"Item": item,
	}).Debug("Incoming Item")

	// We don't have this item queued
	if q.uniqueIDs[id] {
		log.WithFields(log.Fields{
			"ID": id,
			"item": item,
		}).Debug("Item is already queued")
		return false
	}

	q.feeder <- item
	q.uniqueIDs[id] = true

	log.WithFields(log.Fields{
		"feeder":    len(q.feeder),
		"uniqueIDs": len(q.uniqueIDs),
	}).Debug("Incoming Queue Lengths")

	return true
}

func (q *UniqueQueue) popTo(sendTo func(interface{})) {
	select {
	case item, feederOk := <-q.feeder:
		if !feederOk {
			log.Debug("Feeder Queue is closed")
			q.feederOk = false
			return
		}

		id := q.getID(item)

		log.WithFields(log.Fields{
			"ID":   id,
			"Item": item,
		}).Debug("Outgoing Item")

		sendTo(item)
		delete(q.uniqueIDs, id) // drop it from the set

		log.WithFields(log.Fields{
			"feeder":    len(q.feeder),
			"uniqueIDs": len(q.uniqueIDs),
		}).Debug("Outgoing Queue Lengths")

	default:
		// We don't want this to block the rest of the loop
	}
}
