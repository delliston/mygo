package lift

import (
	"fmt"
	"time"
	"log"
)

/*
	Elevator Algorithm:
	If going UP
		Keep going UP while there exists above either a dropoff or an (UPward) pickup.
			Do not service DOWNward pickups. This elevator is going UP.
		When no dropoffs or UPward pickups above,
			doHighestDownwardPickup:
				pickup := highest DOWNward pickup
				dir := dirTo(pickup)
				heading := DOWN        // do we need this???
				go to pickup
				While moving
					If higher DOWNard pickup, go there first.
					Service dropoffs on route.
					Do not service UPward pickups (except at dropoffs)
					Do not service DOWNward pickups below.
		OPTION: Reject any downward dropoff when elevator is going up.

	If IDLE:
		on dropoff (e.g., DOWNward)
			dir := dirTo(dropoff) // say, DOWN.
			Go towards dropoff and for floors en route:
				Service any new dropoffs or DOWN pickups.
				Do not service any upward dropoffs or pickups
		on pickup: (e.g., above and DOWNward)
			doTopDownwardPickup() -- defined above.

	If going DOWN, reverse the UP clause.

	OLD: Simpler Algorithm
	  - @STEP
	  	while direction != 0:
		  	floor += direction
		  	if floor in pickups or dropoffs:
		  		open door, delete pickup & dropoff
		  	else if no pickups/dropoffs above:
		  		if pickups/dropoffs below:
		  			direction *= -1
		  		else:
		  			direction = 0

	CASE: An IDLE elevator at floor 1 receives a pickup DOWN from floor 10. It heads toward 10.
		  If it gets an UP request from floor 15, en route, it will not service it before taking 10 DOWN.
		  If pickup DOWN @ 10 is serviced, and passenger requests dropoff above, what should we do?

	CASE: An IDLE elevator receives a pickup request for the floor it is on. Yep, not handled.

	USE CASES
	- A pickup is issued to an elevator A. Later, elevator B stops at that floor, going in the right direction.
	  Then the first pickup should be cancelled.
	- How to prevent starvation? Should pickup requests be handled in the order received?
*/

// Keeps track of those waiting for an Arrival
type ArrivalListeners map[FloorDir][]chan<- Arrival		// Tracks for each FloorDir
func (m ArrivalListeners) addDropoffListener(dropoff Dropoff) {
	m._addListener(FloorDir{dropoff.Floor, IDLE}, dropoff.Done)
}
func (m ArrivalListeners) addPickupListener(pickup Pickup) {
	m._addListener(FloorDir{pickup.Floor, pickup.Dir}, pickup.Done)
}
func (m ArrivalListeners) _addListener(floorDir FloorDir, listener chan<- Arrival) {
	arr := m[floorDir]
	if arr == nil {
		arr = make([]chan<- Arrival, 0)
	}
	arr = append(arr, listener)
	m[floorDir] = arr
}
func (m ArrivalListeners) notifyArrival(arrival Arrival) {
	// Notify dropoffs.
	m._notify(FloorDir{arrival.Floor, IDLE}, arrival)

	// Notify pickups iff we have a direction.
	if arrival.Dir != IDLE {
		// Notify pickups
		m._notify(FloorDir{arrival.Floor, arrival.Dir}, arrival)
	}
}

// Notifies the Pickup and Dropoff listeners.
func (m ArrivalListeners) _notify(floorDir FloorDir, arrival Arrival) {
	arr := m[floorDir]
	if arr != nil {
		for _, ch := range arr {
			go func() {
				ch<- arrival		// FUTURE: Handle closed channel
			}()
		}
		m[floorDir] = nil
	}
}


// Elevator implements Conveyor:
// 	- Reads Pickups from a channel.
//	- Reads Dropoffs from a channel.
//	- Writes Arrivals to a channel.
// Internally, it stores
//		dropoff [floorNum] (request issued inside elevator by passenger)
//		pickup [floorNum] (request issued outside elevator by potential passenger)
//	It is not possible to cancel a dropoff request (as in real elevators).
type Elevator struct {
	id int
	numFloors int
	floor Floor  	// The last floor we passed, or (if dir==IDLE, the floor we are sitting on). The floor has already been serviced.
	dest Floor		// The current destination. dir == floor.DirectionTo(dest). If dir == IDLE, dest == floor.
	dir Direction   // Current direction of the elevator: UP, DOWN, or IDLE.
	pickup *Pickup   // Special: The pickup we are rushing towards; after arriving we will set dir = pickup.Dir
	dropoffs *FloorSet    // Which dropoffs (destinations) are requested
	pickupsUp *FloorSet   // Which pickups (origins) are requested UP
	pickupsDown *FloorSet   // Which pickups (origins) are requested DOWN
	chPickups chan Pickup // System sends us pickup demands
	chDropoffs chan Dropoff // System (or Passenger) sends us dropoff requests from inside elevator.
	chArrivals chan Arrival // We send when we arrive at a floor (in a direction). FUTURE: Should send dir=IDLE if no outstanding reqs.
	waiters ArrivalListeners
	drive *elevatorDriver
	//	chPickupQueries chan Pickup // System asks for pickup estimates
	//	chPickupQueryEstimates chan PickupEstimate // System asks for pickup estimates
}
func NewElevator(id int, numFloors int) (*Elevator) {
	e := &Elevator{ id, numFloors, 0, 0, IDLE, nil,
					newFloorSet(numFloors), newFloorSet(numFloors), newFloorSet(numFloors),
					make(chan Pickup), make(chan Dropoff), make(chan Arrival),
					make(ArrivalListeners), newDriver(id)}
	go e.mainLoop()
	return e
}
//func (e *Elevator) PickupQueries() chan<- Pickup { return e.chPickupQueries }
//func (e *Elevator) PickupQueryEstimates() chan<- PickupEstimate { return e.chPickupQueryEstimates }
func (e *Elevator) Id() int { return e.id }
func (e *Elevator) Pickups() chan<- Pickup { return e.chPickups }
func (e *Elevator) Dropoffs() chan<- Dropoff { return e.chDropoffs }
func (e *Elevator) Arrivals() <-chan Arrival { return e.chArrivals }

// Passenger inside elevator punches a floor button
func (e *Elevator) pickups(dir Direction) *FloorSet {
	if dir == UP {
		return e.pickupsUp
	} else if dir == DOWN {
		return e.pickupsDown
	} else {
		panic(fmt.Sprintf("invalid direction for pickups: %d", dir))
	}
}

func (e *Elevator) gotoPickup(pickup Pickup) {
	e.pickup = &pickup
	e.dest = pickup.Floor
	e.dir = e.floor.DirectionTo(pickup.Floor)
	e.sendDriveTo(pickup.Floor) // updates e.dir, e.nextStop
}

func (e *Elevator) gotoFloor(floor Floor) {
	e.pickup = nil
	e.dest = floor
	e.dir = e.floor.DirectionTo(floor)
	e.sendDriveTo(floor) // updates e.dir, e.nextStop
}

// Sends cmd to Drive to set floor, and checks response. If accepted, updates e.dir and e.nextStop
func (e *Elevator) sendDriveTo(dest Floor) {
	chReply := make(chan Floor)
	log.Printf("Elevator-%d sending drive to %v", e.id, dest)
	e.drive.chRequests<- DriverDestRequest{dest, chReply}
	e.dest = <-chReply
	e.dir = e.floor.DirectionTo(e.dest)	// whether the new e.dest is the specified dest, or if it failed, stil udpate dir.
	log.Printf("Elevator-%d drive replied with dest %v", e.id, e.dest)
}


func (e *Elevator) mainLoop() {
	for {
		select {
//		case pickup := <-e.chPickupQueries:
//			// Passenger outside elevator requests pickup System Request for pickup proposal.
//			// Return #stops/distance before this pickup
//			e.onPickupQuery(pickup)    // FUTURE: Handle proposals.

		case pickup := <-e.chPickups:
			// Passenger outside elevator requests pickup. System assigns request to us.
			e.onPickupReq(pickup)

		case dropoff := <-e.chDropoffs:
			// Passenger inside elevator requests dropoff
			e.onDropoffReq(dropoff)

		case s := <-e.drive.chNotifications:
			// ElevatorDrive has passed or stopped at a floor
			e.onDriveNotification(s)
		}
	}
}

/*
func (e *Elevator) onPickupQuery(pickup Pickup) {
	stopsUntilPickup := 0
	distanceUntilPickup := 0
	goinThereAnyway := false
	// TODO: Fill in these values
	e.chPickupQueryEstimates <- &PickupEstimate{ pickup, stopsUntilPickup, distanceUntilPickup, goinThereAnyway }
}
*/

func (e *Elevator) onPickupReq(pickup Pickup) {
	log.Printf("Elevator-%d received req %v\n", e.id, pickup)
	e.waiters.addPickupListener(pickup)
	if !e.pickups(pickup.Dir).set(pickup.Floor) {    // returns previous value. If it wasn't set before, then:
		// Decide whether to go to the new pickup instead.
		if e.dir == IDLE {
			// 1. We are IDLE
			e.gotoPickup(pickup)
		} else if e.pickup != nil {
			// 2. We are rushing (in direction e.dir) to a pickup AND either:
			//    A) The new pickup lies between e.Floor and e.pickup.Floor
			//	     AND e.dir == pickup.Dir   // after pickup, we continue beyond it
			//    B) The new pickup lies beyond e.pickup.Floor (in direction e.dir) from e.floor
			//	     AND e.dir == opposite of (pickup.Dir) // after pickup, we switch direction
			if pickup.Floor.between(e.floor, e.pickup.Floor) {
				if e.dir == pickup.Dir {
					e.gotoPickup(pickup)
				}
			} else {
				if e.dir == pickup.Dir.opposite() {
					e.gotoPickup(pickup)
				}
			}
		} else {
			// 3. This pickup lies en route to and in the direction of to our current destination.
			if e.dir == pickup.Dir && pickup.Floor.between(e.floor, e.dest) {
				e.gotoFloor(pickup.Floor)
			}
		}
	}
}

func (e *Elevator) onDropoffReq(dropoff Dropoff) {
	log.Printf("Elevator-%d received req %v\n", e.id, dropoff)
	e.waiters.addDropoffListener(dropoff)
	if !e.dropoffs.set(dropoff.Floor) {    // returns previous value
		if e.dir == IDLE {
			e.gotoFloor(dropoff.Floor)
		} else if e.pickup != nil {
			// We ignore the dropoff request. We are rushing to a pickup.
			// Ouch, that's bad if the pickup comes when user is entering elevator and selecting a floor.
			// FUTURE: For some period (doors open), block pickup requests, only accept dropoff requests.
		} else if dropoff.Floor.between(e.floor, e.dest) {
			e.gotoFloor(dropoff.Floor)
		}
	}
}

func (e *Elevator) onDriveNotification(s DriverStopNotification) {
	e.floor = s.floor
	if s.stopping {
		if e.pickup != nil && e.pickup.Floor == e.floor {
			// We've hit our target pickup. We may switch direction.
			e.dir = e.pickup.Dir
			e.pickup = nil
		}
		e.pickups(e.dir).clear(e.floor)        // FUTURE: signal correct pickup light to clear.
		e.dropoffs.clear(e.floor)

		arrival := Arrival{ e.floor, e.dir, e }
		e.waiters.notifyArrival(arrival)     // Notifies all waiters

		// TODO: Passengers we just picked up have not entered their desired stop. Therefore, we shouldn't pick our next stop yet?

		// Determine next stop
		dest := findNext(e.floor, e.dir, e.dropoffs, e.pickups(e.dir)) // returns first arg if no place to go
		if dest == e.floor {
			// TODO: We should check for dropoffs or pickups in the other direction.
			e.dir = IDLE
		} else {
			e.gotoFloor(dest)
		}
	}
}

// Find the next Floor (in the direction), among the specified FloorSets.
// If none, return cur.
func findNext(cur Floor, direction Direction, floorSets... *FloorSet) Floor {
	next := cur
	for _, fs := range floorSets {
		fsNext := fs.next(cur, direction)
		if fsNext != cur && fsNext < next {
			next = fsNext
		}
	}
	return next
}



// elevatorDriver receives requests to stop at a particular floor.
// It knows its direction, last floor (passed if dir!=IDLE) and destination floor.
// On request:
// It's not possible to change the elevator's direction when it's underway to a destination.
// However, it is allowed to change the destination to a floor between the last floor and the dest. (stopping short).
// On request, if it can stop at the new floor, it sets it as new destination and returns this value.
// If it cannot (because it has passed the floor, or (FUTURE) because it is approaching the floor too fast to stop),
// then a request does not change its destination, and returns the old value.
type elevatorDriver struct {
	id int
	floor Floor			// Current floor (if dir == IDLE), else last floor passed (and we are DIR - above/below it
	dest Floor			// Destination. if dir == IDLE, then floor == dest.
	dir Direction		// If IDLE, not moving. Always, dir == floor.directionTo(dest).
	chRequests chan DriverDestRequest
	chNotifications chan DriverStopNotification
}
func newDriver(id int) *elevatorDriver {
	d := &elevatorDriver{ id, 0, 0, IDLE, make(chan DriverDestRequest), make(chan DriverStopNotification) }
	go d.mainLoop(0)
	return d
}

type DriverDestRequest struct {
	floor Floor
	chReply chan<- Floor
}

type DriverStopNotification struct {
	floor Floor
	stopping bool
}

func (d *elevatorDriver) mainLoop(floor Floor) {
	var timer <-chan time.Time
	for {
		select {
		case req := <-d.chRequests:
			// Request to set/change destination.
			if d.dir == IDLE {	// then floor == dest
				if d.floor != req.floor {
					d.dest = req.floor
					d.dir = d.floor.DirectionTo(d.dest)
					// start moving
					timer = time.After(TimeBetweenFloors) // FUTURE: set speed
					log.Printf("Elevator-%d at %s going %s to %s\n", d.id, d.floor, d.dir, d.dest)
				}
			} else if req.floor.between(d.floor, d.dest) {
				// New floor is on route to current dest.
				// FUTURE: check if we can slow down in time. Reduce speed if needed.
				log.Printf("Elevator-%d going %s changed destination from %s to %s\n", d.id, d.id, d.dir, d.dest, req.floor)
				d.dest = req.floor
			} else if floor.DirectionTo(req.floor) == d.dir {
				// New floor is beyond our current dest in the same direction. Go there.
				// FUTURE: Increase speed if needed.
				log.Printf("Elevator-%d going %s changed destination from %s to %s\n", d.id, d.dir, d.dest, req.floor)
				d.dest = req.floor
			}
			req.chReply<- d.dest

		case <-timer:
			// Passing or stopping at a floor.
			d.floor = d.floor.next(d.dir)	// I.e.: d.floor += d.dir
			if d.floor == d.dest {
				log.Printf("Elevator-%d stopped at %d\n", d.id, d.floor)
				d.dir = IDLE	// stop
				timer = nil
			} else {
				log.Printf("Elevator-%d passing %s %s\n", d.id, d.floor, d.dir)
				timer = time.After(TimeBetweenFloors)
			}
			d.chNotifications<- DriverStopNotification{d.floor, d.floor == d.dest}
		}
	}
}

/*
func (e *Elevator) calcNextStop() {
	// When we are called, e.floor has already been serviced.

	// If we are moving UP or DOWN, find the next pickup or dropoff in that direction
	if e.direction == UP {
		next := findNext(e.floor, e.direction, e.pickupsUp, e.dropoffs)
		if next == e.floor { // None found
			next := e.pickupsDown.highest(e.floor)
		}
	} else if e.direction == DOWN {
		// TODO: as with UP
		next := findNext(e.floor, e.direction, e.pickupsDown, e.dropoffs)
		if next == e.floor { // None found
			next := e.pickupsUp.lowest(e.floor)
		}
	} else {	// e.direction == IDLE
		// There are no drop
		next := 
	}
}
*/
