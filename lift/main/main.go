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
	NumFloors := 20	// Floors are numbered from 0
	NumElevators := 3	// TODO: Read these from args
	s := lift.NewSystem(NumFloors, NumElevators)

	wgPass := sync.WaitGroup{}
//	wgPass.Add(1)		// hack

//  	go func() {
		for id := 1; id < 3; id++ {
			wgPass.Add(1)
			p := &Passenger{ id, lift.Floor(rand.Intn(NumFloors)), lift.Floor(rand.Intn(NumFloors)) }
			log.Printf("Pass-%d: created with start %s, dest %s\n", id, p.start, p.dest)
			go func() {
				p.mainLoop(s.PickupReqs())
				wgPass.Done()
			}()
			time.Sleep(5 * lift.Tick)
		}
//		wgPass.Done()	// hack
//	}()

	wgPass.Wait()	// Waits until all passengers complete. This is a bit random. May exit immediately if first passenger has src=dest.
}

// Passenger is group of people who requests a pickup or dropoff
// NOT USED.
type Passenger struct {
	id int
	start lift.Floor
	dest lift.Floor
	// FUTURE: Passenger should not specify destination (until picked up).
	// So, specify a channel it will read (when picked up) which accepts a writable chan of destination
	// E.g.:
	// chan chan<- Floor
}

func (p *Passenger) mainLoop(chPickupReqs chan<- lift.PickupReq) {
	if p.start == p.dest {
		fmt.Printf("Pass-%d skipping elevator: start %s == dest %s", p.start, p.dest)
		return
	}
	// FUTURE: sleep random time

	chChDropoffs := make(lift.ArrivalChannel)
	dir := p.start.DirectionTo(p.dest)
	log.Printf("Pass-%d requesting pickup %s %s\n", p.id, p.start, dir)
	chPickupReqs <- lift.PickupReq{lift.NewPickup(p.start, dir), chChDropoffs}
	log.Printf("Pass-%d waiting for pickup %s %s on channel %v\n", p.id, p.start, dir, chChDropoffs)

	// Wait until an arrival, indicated by receiving a chan<- Dropoff
	chDropoff := <-chChDropoffs
	log.Printf("Pass-%d got lift %s %s\n", p.id, p.start, dir)

	// Board and press button
	time.Sleep(lift.TimeSelectDropoff)
	log.Printf("Pass-%d requesting dropoff %s\n", p.id, p.dest)
	chDropoff <- lift.NewDropoff(p.dest)
	// TODO	chDropoff <- lift.DropoffReq{lift.NewDropoff(p.dest), chChDropoffs()}	// shouldn't reuse that channel

	log.Printf("Pass-%d riding to floor %s\n", p.id, p.dest)


	// TODO: Wait until dropoff completed (arrival). Shouldn't that come from elevator?
	//	Change Dropoff to DropoffReq{ p.dest, chan<- Arrival }
}


