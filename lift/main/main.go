package main

import (
	"github.com/delliston/mygo/lift"
	"time"
	"math/rand"
	"sync"
	"log"
	"fmt"
)


// This could become a System type
func main() {
	NumFloors := 10	// Floors are numbered from 0
	NumElevators := 3	// TODO: Read these from args
	s := lift.NewSystem(NumFloors, NumElevators)

	wgPass := sync.WaitGroup{}

	for id := 1; id < 5; id++ {
		wgPass.Add(1)
		p := &Passenger{ id, lift.Floor(rand.Intn(NumFloors)), lift.Floor(rand.Intn(NumFloors)) }
		log.Printf("Pass-%d: created with start %s, dest %s\n", id, p.start, p.dest)
		go func() {
			p.main(s.Pickups())
			wgPass.Done()
		}()
		time.Sleep(5 * lift.Tick)
	}
	wgPass.Wait()	// Waits until all passengers complete. This is a bit random. May exit immediately if first passenger has src=dest.
}

// Passenger is group of people who requests a pickup or dropoff
// NOT USED.
type Passenger struct {
	id int
	start lift.Floor	// Ick: naming start v. end, origin vs. dest?
	dest lift.Floor
}

func (p *Passenger) main(chPickupReqs chan<- lift.Pickup) {
	if p.start == p.dest {
		fmt.Printf("Pass-%d skipping elevator: start %s == dest %s", p.start, p.dest)
		return
	}

	// Request pickup and wait.
	chArrival := make(chan lift.Arrival)
	dir := p.start.DirectionTo(p.dest)
	pickup := lift.Pickup{p.start, dir, chArrival}
	log.Printf("Pass-%d requesting pickup %s %s\n", p.id, p.start, dir)
	chPickupReqs <- pickup
	log.Printf("Pass-%d waiting for pickup %s %s on channel %v\n", p.id, p.start, dir, chArrival)

	// Wait for arrival.
	a := <-chArrival
	if a.Floor != p.start {
		panic(fmt.Sprintf("Waiting at %s, but pickup arrival says %s", p.start, a.Floor))
	}
	if a.Dir != dir {
		panic(fmt.Sprintf("Waiting for %s lift, but pickup arrival says direction is %s", dir, a.Dir))
	}

	// Board and press button.
	log.Printf("Pass-%d boarded Elevator-%d at %s %s\n", p.id, a.Conveyor.Id(), p.start, dir)
	time.Sleep(lift.TimeSelectDropoff)	// FUTURE: elevator door may close before passenger boards.
	log.Printf("Pass-%d requesting dropoff %s\n", p.id, p.dest)
	dropoff := lift.Dropoff{ p.dest, chArrival }	// FUTURE: May want to use different channel for pickup and dropoff
	a.Conveyor.Dropoffs() <- dropoff
	log.Printf("Pass-%d riding to floor %s\n", p.id, p.dest)

	// Wait for arrival
	a = <-chArrival
	if a.Floor != p.dest {
		panic(fmt.Sprintf("Pass-%d waiting to arrive at at %s, but dropoff arrival says %s", p.dest, a.Floor))
	}
	log.Printf("Pass-%d arrived at destination floor %s\n", p.id, p.dest)
}


