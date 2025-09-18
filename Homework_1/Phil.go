package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	// n = number of Philosophers
	n               = 5
	eatingGoal      = 3
	maxThinkingTime = 3 * time.Second
	maxEatingTime   = 1 * time.Second
)

// give: fork sends once to give permission to take
// release: philosoph sends when releasing forks
// philosophID : self-explanatory
type forkRequest struct {
	philosopherId int
	give, release chan struct{}
}

// fork runs it own goRoutine
// It works on a FIFO-order
// Only works through channels
func fork(_ int, requests <-chan forkRequest) {
	for req := range requests {
		// grant exclusive use
		req.give <- struct{}{}
		// wait for release
		<-req.release
		// next request
	}
}

type Philosopher struct {
	id                int
	leftForkRequests  chan<- forkRequest
	rightForkRequests chan<- forkRequest
	eatNum            int
}

func (p Philosopher) getFork(reqCh chan<- forkRequest) (give, release chan struct{}) {
	give = make(chan struct{})
	release = make(chan struct{})
	req := forkRequest{philosopherId: p.id, give: give, release: release}
	// send request: this blocks if the fork is in use
	reqCh <- req
	// wait until the fork gives permission
	<-give
	return
}

func (p Philosopher) eat(eaten int, wg *sync.WaitGroup) {

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	defer wg.Done()
	for eaten < p.eatNum {
		// thinking
		fmt.Printf("Philosopher %d: thinking\n", p.id)
		time.Sleep(time.Duration(r.Int63n(int64(maxThinkingTime))))

		//how this avoids deadlock
		//odd philosophers take left, then right.
		//even philosophers take right, then left.
		//there is never circular wait then.

		var first, second chan<- forkRequest
		if p.id%2 == 1 {
			first, second = p.leftForkRequests, p.rightForkRequests
		} else {
			first, second = p.rightForkRequests, p.leftForkRequests
		}

		//take fork in order through channels
		_, rel1 := p.getFork(first)
		_, rel2 := p.getFork(second)

		// eating
		fmt.Printf("Philosopher %d: eating (meal %d)\n", p.id, eaten+1)
		time.Sleep(time.Duration(r.Int63n(int64(maxEatingTime))))

		//release forks
		rel2 <- struct{}{}
		rel1 <- struct{}{}

		eaten++
		//wait a little bit before trying to eat again
		time.Sleep(time.Duration(r.Intn(200)) * time.Millisecond)
	}

	fmt.Printf("Philosopher %d: finished eating (ate %d times)\n", p.id, eaten)
}

func main() {
	var wg sync.WaitGroup
	wg.Add(n)
	// Create fork request channels and goRoutines
	forkCh := make([]chan forkRequest, n)
	for i := 0; i < n; i++ {
		forkCh[i] = make(chan forkRequest)
		go fork(i, forkCh[i])
	}

	// goroutines for Philosophers
	for i := 0; i < n; i++ {
		p := Philosopher{
			id:                i,
			leftForkRequests:  forkCh[i],
			rightForkRequests: forkCh[(i+1)%n],
			eatNum:            eatingGoal,
		}
		go p.eat(0, &wg)
	}
	wg.Wait()

	fmt.Println("All Philosophers are done! and have eaten 3 times :D")
}
