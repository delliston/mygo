package lift

import (
	"fmt"
	"time"
	"strconv"
)

// System Time (at least for simulation).
const(
	Tick = 100 * time.Millisecond
	TimeServiceFloor = 25 * Tick
	TimeBetweenFloors = 10 * Tick
	TimeSelectDropoff = 15 * Tick
)

// The floors start at zero
type Floor int

func (f Floor) String() string { return strconv.Itoa(int(f)) }
func (f Floor) between(f1, f2 Floor) bool {
	return f1 < f && f < f2
}

func (f Floor) next(dir Direction) Floor {
	return Floor(int(f)+int(dir))
}

func (f Floor) DirectionTo(dest Floor) Direction {
	if f == dest {
		return IDLE
	} else if dest > f {
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

func (d Direction) String() string {
	switch d {
	case UP:
		return "UP"
	case DOWN:
		return "DOWN"
	case IDLE:
		return "IDLE"
	default:
		panic(fmt.Sprintf("Unknown direction: %d", d))
	}
}

// Not sure I like this extraction of common elements of Arrival and Pickup.
type FloorDir struct {
	floor Floor
	dir Direction
}

type Arrival FloorDir
func (a Arrival) String() string {
	return fmt.Sprintf("Arrival(%s %s)", a.floor, a.dir)
}

// Requests: Dropoff, Pickup, PickupQuery.

type Pickup FloorDir
func NewPickup(floor Floor, dir Direction) (Pickup) { return Pickup{floor, dir} }	// Bad, NewX should return *X. Really want Pickup.ValueOf(floor, dir).
func (p Pickup) String() string {
	return fmt.Sprintf("Pickup(%s %s)", p.floor, p.dir)
}


type Dropoff struct {
	floor Floor
}
func NewDropoff(floor Floor) (Dropoff) { return Dropoff{floor}}

// Used by System (not Elevator). The pickup is acknowledged by sending a writable channel for specifying the Dropoff.
type PickupReq struct {
	Pickup    // The Pickup coordinates
	Done ArrivalChannel  // That is, a readable channel of writable channel of Dropoff. Can interfaces help make this understandable?
}
func (p PickupReq) String() string {
	return fmt.Sprintf("PickupReq(%s %s)", p.floor, p.dir)
}

// Used by System (not Elevator). The dropoff is acknowledged by sending a writable channel for specifying the Dropoff. Just reusing PickupReq types here, not sure why I would want to receive the drop
type DropoffReq struct {
	Dropoff    // The Dropoff floor
	Done chan interface{}	// Signalled when Dropoff completes
}
func (d DropoffReq) String() string {
	return fmt.Sprintf("DropoffReq(%s)", d.floor)
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
	Dropoffs() chan<- Dropoff
	Arrivals() <-chan Arrival	// Readable channel: alerts
	// FUTURE: DropoffsDone() <-chan Dropoff  // not so much needed
	// FUTURE: PickupCancellations() chan<- Pickup -- system invokes when another elevator makes the pickup,
		//		e.g., implicitly when making a dropoff
}
