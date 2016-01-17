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

Create a struct-literal:

```
inItems := []String {"0123", "4567", "89ab", "cdef"}

uq = (&queue.UniqueQueue{
    MatcherID:   func(incoming interface{}) string {
      return incoming.(String)
    },
    QueueLength: 1
    BufferSize:  10,
}).Init()

uq.Run()

for _, item := range inItems {
    uq.In <- item
}

for item := range uq.Out {
    fmt.Printf("%v", item)
}
