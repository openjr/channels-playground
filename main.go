package main

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"
)

type InputMessage struct {
	ctx             context.Context
	cancelFunc      context.CancelFunc
	responseChannel chan int
}

func NewInputMessage(d time.Duration) InputMessage {
	ctx, cancelFunc := context.WithTimeout(context.Background(), d)
	return InputMessage{
		// make sure the channel is buffered, in case we time out
		responseChannel: make(chan int, 1),
		ctx:             ctx,
		cancelFunc:      cancelFunc,
	}
}

func main() {
	inputChannel := make(chan InputMessage)

	go worker(inputChannel)

	var wg sync.WaitGroup
	amountOfWorkers := 5

	wg.Add(amountOfWorkers)
	for i := 0; i < amountOfWorkers; i++ {
		go generator(i, inputChannel, &wg)
	}
	wg.Wait()
	close(inputChannel)
}

func generator(id int, inputChannel chan InputMessage, wg *sync.WaitGroup) {
	inputMessage := NewInputMessage(500 * time.Microsecond)
	defer inputMessage.cancelFunc()
	defer wg.Done()

	select {
	// we reached the deadline and couldn't send the message, bail
	case <-inputMessage.ctx.Done():
		log.Println("[Generator]:", id, "Dropped message, worker is busy:", inputMessage.ctx.Err())

	// we managed to send the message to the worker
	case inputChannel <- inputMessage:
		select {
		// we can receive the response before we run out of time
		case v := <-inputMessage.responseChannel:
			log.Println("[Generator]:", id, "value:", v)

		// took us too long to get the response back
		case <-inputMessage.ctx.Done():
			log.Println("[Generator]:", id, "timed out, error:", inputMessage.ctx.Err())
		}
	}
}

func worker(inputC chan InputMessage) {
	var value int
	for msg := range inputC {
		// fake operation that takes long
		i := rand.Intn(500)
		randomDuration := time.Duration(i) * time.Microsecond
		time.Sleep(randomDuration)

		select {
		// try to send the response, if we don't get an answer just bail
		case msg.responseChannel <- value:
			value++
		case <-msg.ctx.Done():
			log.Println("timeout, not doing anything")
		}

		// we only have one writer to the channel, close it after we are done with the message
		close(msg.responseChannel)

	}
}
