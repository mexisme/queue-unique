# queue-unique

A relatively simple queuing library for Golang based on channels that only pushes values that are
not already in the queue

Items passed to the `In` queue are buffered then sent to the `Out` queue.
Any incoming items that are already in the buffer (but haven't made it to the `Out` queue) are
silently dropped.

## Use case

My main use-case for this is being able to pass actions to a Worker Pool, and to make sure an
action is only performed once per "cycle" through the queue.

## Example

```
inItems := []String {"0123", "4567", "89ab", "cdef", "0123", "4567"}

// Create struct-literal
uq = (&queue.UniqueQueue{
    MatcherID:   func(incoming interface{}) string {
      return incoming.(String)
    },
    QueueLength: 1
    BufferSize:  10,
}).Init()

// Start the background FIFO goroutine
uq.Run()

// Put some items into the queue
for _, item := range inItems {
    uq.In <- item
}

// Read some items from the queue
for item := range uq.Out {
    fmt.Printf("%v", item)
}
