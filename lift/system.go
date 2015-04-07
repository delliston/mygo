package lift

import (
	"math/rand"
	"fmt"
	"log"
)

// On arrival, the waiter receives (on this Channel) a channel to which he can send a dropoff request.
// I think this is idiomatic Go, but it looks complicated.
type ArrivalChannel chan chan<- Dropoff

type elevatorArrival struct {
	arrival Arrival
	dropoffReqs chan<- Dropoff
}

// The System provisions the elevators (TODO: structs or channels)
// plus the records of any pickup requests (up or down) at each floor.
type System struct {
	elevators []Conveyor
	pickupsUp *FloorSet // Floors which have outstanding UP requests are true
	pickupsDown *FloorSet // Floors which have outstanding DOWN requests are true
	chPickupReqs chan PickupReq  // System receives Pickup Requests from users.
	chArrivals chan elevatorArrival     // System receives Arrival (notifications) from elevators, sends elevator.Dropoffs() to each PickupReq.done
	pickupsToReplies map[FloorDir][] ArrivalChannel  // On arrival at FloorDir, forward Dropoff channel to all in list.
}
func (s *System) PickupReqs() chan<- PickupReq { return s.chPickupReqs }
func NewSystem(numFloors, numElevators int) *System {
	chElevatorArrivals := make(chan elevatorArrival)
	elevators := make([]Conveyor, numElevators)
	for i := 0; i < numElevators; i++ {
		e := NewElevator(i, numFloors)
		elevators[i] = e
		go func() {	// Forward arrival messages from elevator to global channel (and include this elevator's Dropoff channel).
			for arrival := range e.Arrivals() {
				chElevatorArrivals <- elevatorArrival{arrival, e.Dropoffs()}
			}
		}()
	}
	s:= &System{ elevators, newFloorSet(numFloors), newFloorSet(numFloors),
				 make(chan PickupReq), chElevatorArrivals, make(map[FloorDir][] ArrivalChannel)}
	go s.mainLoop()
	return s
}

func (s *System) pickups(dir Direction) *FloorSet {
	switch dir {
	case UP:
		return s.pickupsUp
	case DOWN:
		return s.pickupsDown
	default:
		panic(fmt.Sprintf("Direction must be up or down, found %d", dir))
	}
}

func (s *System) mainLoop() {
	// Assumptions: all elevators are at floor 0, and all buttons are cleared
	for {
		select {
			case pickupReq := <-s.chPickupReqs:
				s.onPickupReq(pickupReq)
			case arrival := <-s.chArrivals:
				s.onArrival(arrival)
		}
	}
}

func (s *System) onPickupReq(pickupReq PickupReq) {
	log.Printf("System got %v\n", pickupReq)
	s.addArrivalListener(FloorDir(pickupReq.Pickup), pickupReq.Done)
	if ! s.pickups(pickupReq.dir).set(pickupReq.floor) {
		// Find a suitable elevator
		// FUTURE: Send PickupQuery to (some/all) elevators and choose the best result.
		// For now, random:
		id := rand.Intn(len(s.elevators))
		e := s.elevators[id]
		log.Printf("System sending %v to Elevator-%d\n", pickupReq, id)
		e.Pickups() <- pickupReq.Pickup
	}
}

// Needed because GoLang lacks generics. In Java, would be Map<FloorDir, Set<ArrivalChannel>>
func (s *System) addArrivalListener(floordir FloorDir, receiver ArrivalChannel) {
	arr := s.pickupsToReplies[floordir]
	if arr == nil {
		arr = make([]ArrivalChannel, 0)
	}
	arr = append(arr, receiver)
	s.pickupsToReplies[floordir] = arr
}

func (s *System) onArrival(arrival elevatorArrival) {
	log.Printf("System got %v\n", arrival.arrival)
//	s.pickups(arrival.dir).clear(arrival.floor)
//	for _, ch := range s.pickupsToReplies[FloorDir(arrival.arrival)] {
//		// Signal all passengers (pickups) waiting on this FloorDir
//		log.Printf("Signalling arrival to receiver %v\n", ch)
//		ch <- arrival.dropoffReqs
//		log.Println("Signalled arrival")
//	}
}
