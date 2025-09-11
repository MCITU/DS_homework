package DS_homeworkpackage

import (
	"fmt"
	"math/rand"
	"time"
)

// give: fork sends once to give permission to take
// release: philosoph sends when releasing forks
// philosophID : self-explanatory
type forkRequest struct {
	philosophId int
	give        chan struct{}
	release     chan struct{}
}

// fork runs it own goRoutine
// It works on a FIFO-order
// Only works through channels
func fork(id int, requests <-chan forkRequest) {
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

func (p Philosopher) eat(done chan<- struct{}) {
	eaten := 0
	random := rand.New(rand.NewSource(time.Now().UnixNano() + int64(p.id)))

	// helper to request/release a single fork via its request channel
	getFork := func(reqCh chan<- forkRequest) (give chan struct{}, release chan struct{}) {
		give = make(chan struct{})
		release = make(chan struct{})
		req := forkRequest{philosophId: p.id, give: give, release: release}

		// send request: this blocks if the fork is in use
		reqCh <- req
		// wait until the fork gives permission
		<-give
		return give, release
	}

	for eaten < p.eatNum {
		// thinking
		fmt.Printf("Philosopher %d: thinking\n", p.id)
		time.Sleep(time.Duration(150+random.Intn(300)) * time.Millisecond)

		//avoid deadlock
		//odd philosophers take left, then right.
		//even philosophers take right, then left.
		//there is not circular wait.

		var first, second chan<- forkRequest
		if p.id%2 == 1 {
			first, second = p.leftForkRequests, p.rightForkRequests
		} else {
			first, second = p.rightForkRequests, p.leftForkRequests
		}
		//take fork in order through channels
		_, rel1 := getFork(first)
		_, rel2 := getFork(second)

		//eating
		fmt.Printf("Philosopher %d: eating (meal %d)\n", p.id, eaten+1)
		time.Sleep(time.Duration(150+random.Intn(300)) * time.Millisecond)

		//release forks
		rel2 <- struct{}{}
		rel1 <- struct{}{}

		eaten++
		//wait a little bit before trying to eat again
		time.Sleep(time.Duration(random.Intn(200)) * time.Millisecond)
	}
	fmt.Printf("Philosopher %d: finished eating (ate %d times)\n", p.id, eaten)
	done <- struct{}{}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	const n = 5
	const eatingGoal = 3

	// Create fork request channels and goRoutines
	forkCh := make([]chan forkRequest, n)
	for i := 0; i < n; i++ {
		forkCh[i] = make(chan forkRequest)
		go fork(i, forkCh[i])
	}

	// Create philosophers with their own goRotines
	done := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		p := Philosopher{
			id:                i,
			leftForkRequests:  forkCh[i],
			rightForkRequests: forkCh[(i+1)%n],
			eatNum:            eatingGoal,
		}
		go p.eat(done)
	}

	//Wait for all to have eating 3 times.
	for i := 0; i < n; i++ {
		<-done
	}
	fmt.Println("All Philosophers are done! and have eaten 3 times :D")
}
