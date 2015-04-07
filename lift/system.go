package lift

import (
	"math/rand"
//	"fmt"
	"log"
)

// The System provisions the elevators (TODO: structs or channels)
// plus the records of any pickup requests (up or down) at each floor.
type System struct {
	elevators []Conveyor
	// FUTURE: Is it really necessary to track pickups{Up,Down}? Elevators do it already. Seems the System wants to know too.
	pickupsUp *FloorSet // Floors which have outstanding UP requests are true
	pickupsDown *FloorSet // Floors which have outstanding DOWN requests are true
	chPickups chan Pickup  // System receives Pickup Requests from users.
//	chArrivals chan Arrival
//	waiters ArrivalListenerss  // On arrival at FloorDir, forward Arrival channel to all registered listeners.
}
func (s *System) Pickups() chan<- Pickup { return s.chPickups }
func NewSystem(numFloors, numElevators int) *System {
	elevators := make([]Conveyor, numElevators)			// <sigh> In Python, these 4 lines would just be a List Comprehension: [ NewElevator(i, numFloors) for i in range(numFloors) ]
	for i := 0; i < numElevators; i++ {
		elevators[i] = NewElevator(i, numFloors)
	}
	s := &System{ elevators, newFloorSet(numFloors), newFloorSet(numFloors), make(chan Pickup) }
	go s.mainLoop()
	return s
}

func (s *System) mainLoop() {
	// Assumptions: all elevators are at floor 0, and all buttons are cleared
	for {
		select {
			case pickupReq := <-s.chPickups:
				s.onPickupReq(pickupReq)
//			case arrival := <-s.chArrivals:			// Currently, we don't subscribe to these.
//				s.onArrival(arrival)
		}
	}
}

func (s *System) onPickupReq(pickupReq Pickup) {
	log.Printf("System got %v\n", pickupReq)
//	s.addArrivalListener(FloorDir(pickupReq.Pickup), pickupReq.Done)
//	if ! s.pickups(pickupReq.dir).set(pickupReq.floor) {
		// Find a suitable elevator
		// FUTURE: Send PickupQuery to (some/all) elevators and choose the best result.
		// For now, random:
		id := rand.Intn(len(s.elevators))
		e := s.elevators[id]
		log.Printf("System sending %v to Elevator-%d\n", pickupReq, id)
		e.Pickups() <- pickupReq
//	}
}

/* FUTURE: As optimization, we should track the set/cleared status of each Floor's UP/DOWN buttons,
   and not dispatch multiple elevators. But let's get the basics working first.
   This seems to require redirecting all Arrivals to System:
    onPickup, store the pickup.done in ArrivalListeners and replace the done channel with one we listen on.
   adding, and having a goroutine */
//func (s *System) pickups(dir Direction) *FloorSet {
//	switch dir {
//	case UP:
//		return s.pickupsUp
//	case DOWN:
//		return s.pickupsDown
//	default:
//		panic(fmt.Sprintf("Direction must be up or down, found %d", dir))
//	}
//}



// Needed because GoLang lacks generics. In Java, would be Map<FloorDir, Set<ArrivalChannel>>
//func (s *System) onArrival(arrival elevatorArrival) {
//	log.Printf("System got %v\n", arrival.arrival)
//	s.pickups(arrival.dir).clear(arrival.floor)
//	for _, ch := range s.pickupsToReplies[FloorDir(arrival.arrival)] {
//		// Signal all passengers (pickups) waiting on this FloorDir
//		log.Printf("Signalling arrival to receiver %v\n", ch)
//		ch <- arrival.dropoffReqs
//		log.Println("Signalled arrival")
//	}
//}
