package lift

import (
	"fmt"
	"strconv"
	"time"
)

// System Time (at least for simulation).
const (
	Tick              = 100 * time.Millisecond
	TimeServiceFloor  = 25 * Tick
	TimeBetweenFloors = 10 * Tick
	TimeSelectDropoff = 15 * Tick
)

// Floors: start at zero.
type Floor int

func (f Floor) String() string { return strconv.Itoa(int(f)) }
func (f Floor) between(f1, f2 Floor) bool {
	return f1 < f && f < f2
}
func (f Floor) next(dir Direction) Floor {
	return Floor(int(f) + int(dir))
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
	UP   Direction = 1
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

// For convenience, we give a type to the Tuple(Floor, Direction), but we don't use it in the model (e.g., PickupReq specifies floor and dir
type FloorDir struct {
	floor Floor
	dir   Direction // May be IDLE, meaning no particular direction -- e.g., when Elevator signals arrival at a floor, but has no outstanding pickups/dropoffs
}

// A request for Pickup. The pickup is acknowledged by sending a writable channel for specifying the Dropoff.
type Pickup struct {
	Floor Floor // The Pickup coordinates
	Dir   Direction
	Done  chan<- Arrival // On arrival at floor/dir, the Arrival is sent via Done.
}

func (p Pickup) String() string {
	return fmt.Sprintf("Pickup(%s %s)", p.Floor, p.Dir)
}
func (p Pickup) FloorDir() FloorDir { return FloorDir{p.Floor, p.Dir} } // for convenience

// Used to request a Dropoff (from inside the Elevator). The dropoff is acknowledged by sending an Arrival.
type Dropoff struct {
	Floor Floor          // The Dropoff floor
	Done  chan<- Arrival // On arrival at floor, the arriving elevator is sent via Done.
}

func (d Dropoff) String() string {
	return fmt.Sprintf("Dropoff(%s)", d.Floor)
}

// Used to signal when a Conveyor arrives at the Floor in the Direction
type Arrival struct {
	Floor    Floor     // The Pickup coordinates
	Dir      Direction // May be IDLE, if the conveyor has no further dropoffs/pickups planned.
	Conveyor Conveyor
}

func (a Arrival) FloorDir() FloorDir { return FloorDir{a.Floor, a.Dir} } // for convenience

type Requestor interface {
	// Returns a channel to which Pickup requests can be sent.
	Pickups() chan<- Pickup
}

// A Conveyor (e.g., Elevator) is represented by a set of channels which the consumer can use:
// 1. to send pickup requests
// 2. to send dropoff requests
type Conveyor interface {
	Requestor // Pickups()

	Id() int // May not be needed

	// Returns a channel to which Dropoff requests can be sent.
	Dropoffs() chan<- Dropoff

	// Returns a channel to which all arrivals are sent.  TODO: Not needed.
	Arrivals() <-chan Arrival

	// FUTURE
	//  PickupCancellations() chan<- Pickup (?) -- when another elevator makes the pickup, the System should cancel it everywhere.
	//	PickupQueries() chan<- FloorDir // Writable channel: query pickup, Conveyance replies with PickupEstimate
	//	PickupQueryEstimates() <-chan PickupEstimate
}

// FUTURE: Choosing the right pickup.
// - Add channel to Conveyor interface for querying pickup estimate.

// Sent to Elevator to request a PickupEstimate. Best offer will be sent a Pickup.
//type PickupQuery struct {
//	pickup Pickup
//	chResp chan<- PickupEstimate
//}

// FUTURE: how to estimate best pickup? Fields suggest some potential values.
/*
type PickupEstimate struct {
	floor Pickup
	stopsUntilPickup int         // How many stops elevator would make before pickup.
	distanceUntilPickup int      // How many floors elevator would travel before pickup.
	goinThereAnyway bool	// True if the pickup is on the way (and in correct direction) to the elevator's
							// current destination
}
*/
