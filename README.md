# queue-unique

A relatively simple queuing library for Golang based on channels that only pushes values that are
not already in the queue

Items passed to the `In` queue are buffered then sent to the `Out` queue.
Any incoming items that are already in the buffer (but haven't made it to the `Out` queue) are
silently dropped.

The godoc-generated docs can be found, here:
https://godoc.org/github.com/mexisme/queue-unique

## Use case

My main use-case for this is being able to pass actions to a Worker Pool, and to make sure an
action is only performed once per "cycle" through the queue.

The reason why this is useful to us is where we're using the Worker Pool to request cached objects from a service, there's no point in
having the same request queued and waiting for a worker multiple times when doing it once will satisfy all the queued requests.

## Example

```
inItems := []String {"0123", "4567", "89ab", "cdef", "0123", "4567"}

// Create struct-literal
uq = (&queue.UniqueQueue{
    MatcherID:   func(incoming interface{}) string {
      return incoming.(String)
    },
    In:          make(queue.InQueue, 100)
    Out:         make(queue.OutQueue, 100)
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
```

## Note

This is a bit of an experiment in replacing both request-response style of code (as you might with a client-server arrangement) and a
mutex (around the Unique ID stack).

As such, the internal mechanisms are highly likely to change over time, and the external API is likely to adjust slightly.

## CONTRIBUTING

- Raise an issue, documenting your planned change
- Fork the code
- Write code, pass tests
- Send a PR
- Profit!

## ToDo's

Take a look at [TODO.org](./TODO.org)
