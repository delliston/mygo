package elevator

import "fmt"

// The floors start at zero
type Floor int

func (f Floor) between(f1, f2 Floor) bool {
	return f1 < f && f < f2
}

func (f Floor) directionTo(dest Floor) Direction {
	if f == dest {
		return IDLE
	} else if f > dest {
		return UP
	} else {
		return DOWN
	}
}

// Direction
type Direction int
const (
	UP Direction = 1
	IDLE Direction = 0
	DOWN Direction = -1
)

func (d Direction) opposite() Direction {
	switch d {
	case UP:
		return DOWN
	case DOWN:
		return UP
	default:
		panic(fmt.Sprintf("Cannot determine opposite of direction %d", d))
	}
}



// Requests: Dropoff, Pickup, PickupQuery.

// Dropoff
type Dropoff struct {
	floor Floor
}
type Pickup struct {
	floor Floor
	direction Direction
}

// Sent to Elevator to request a PickupEstimate. Best offer will be sent a Pickup.
//type PickupQuery struct {
//	pickup Pickup
//	chResp chan<- PickupEstimate
//}

// FUTURE: how to estimate best pickup? Fields suggest some potential values.
/*
type PickupEstimate struct {
	pickup Pickup
	stopsUntilPickup int         // How many stops elevator would make before pickup.
	distanceUntilPickup int      // How many floors elevator would travel before pickup.
	goinThereAnyway bool	// True if the pickup is on the way (and in correct direction) to the elevator's
							// current destination
}
*/

// A Conveyor (e.g., Elevator) is represented by a set of channels which the consumer can use:
// 1. to send pickup queries & demands
// 2. to receive pickup estimates & completions (corresponding to the requests).
type Conveyor interface {
//	PickupQueries() chan<- Pickup // Writable channel: query pickup, Conveyance replies with PickupEstimate
//	PickupQueryEstimates() <-chan PickupEstimate
	Pickups() chan<- Pickup
	PickupsDone() <-chan Pickup	// Readable channel: alerts
	Dropoffs() chan<- Dropoff
	// FUTURE: DropoffsDone() <-chan Dropoff  // not so much needed
	// TODO: PickupCancellations() chan<- Pickup -- system invokes when another elevator makes the pickup.
}



// Passenger is group of people who requests a pickup or dropoff
// NOT USED.
type Passenger struct {
	pickup Pickup
	dest Floor
	// FUTURE: Passenger should not specify destination (until picked up).
	// So, specify a channel it will read (when picked up) which accepts a writable chan of destination
	// E.g.:
		// chan chan<- Floor
}


