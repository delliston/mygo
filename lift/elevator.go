package lift

import (
	"fmt"
	"log"
)

/*
	Elevator Algorithm (concept - implementation differs)
	If going UP
		Keep going UP while there exists ABOVE either a dropoff or an (UPward) pickup.
			Do not service DOWNward pickups. This elevator is going UP.
		When no dropoffs or UPward pickups above,
			doHighestDownwardPickup:
				pickup := highest DOWNward pickup at or above.
				dir := dirTo(pickup)
				heading := DOWN        // do we need this???
				go to pickup
				While moving
					If higher DOWNard pickup, go there first.
					Service dropoffs on route.
					Do not service UPward pickups (except at dropoffs)
					Do not service DOWNward pickups below.
		See code for further cases:
		OPTION: Reject any downward dropoff when elevator is going up.

	If IDLE:
		on dropoff (e.g., DOWNward)
			dir := dirTo(dropoff) // say, DOWN.
			Go towards dropoff and for floors en route:
				Service any new dropoffs or DOWN pickups.
				Do not service any upward dropoffs or pickups
		on pickup: (e.g., above and DOWNward)
			doHighestDownwardPickup() -- defined above.

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

	FUTURE - Some cases not handled yet.
	- An IDLE elevator at floor 1 receives a pickup DOWN from floor 10. It heads toward 10.
		If it gets an UP request from floor 15, en route, it will not service it before taking 10 DOWN.
	 	If pickup DOWN @ 10 is serviced, and passenger requests dropoff above, what should we do?
	- An IDLE elevator receives a pickup request for the floor it is on. Yep, not handled.
	- A pickup is issued to an elevator A. Later, elevator B stops at that floor, going in the right direction.
		Then the first pickup should be cancelled.
	- How to prevent starvation? Should pickup requests be handled in the order received?
*/

// Elevator implements Conveyor:
// 	- Reads Pickups from a channel.
//	- Reads Dropoffs from a channel.
//	- Writes Arrivals to a channel.
// Internally, it stores
//		dropoff [floorNum] (request issued inside elevator by passenger)
//		pickup [floorNum] (request issued outside elevator by potential passenger)
//	It is not possible to cancel a dropoff request (as in real elevators).
type Elevator struct {
	id          int
	numFloors   int
	floor       Floor        // The last floor we passed, or (if dir==IDLE, the floor we are sitting on). The floor has already been serviced.
	dest        Floor        // The current destination. dir == floor.DirectionTo(dest). If dir == IDLE, dest == floor.
	dir         Direction    // Current direction of the elevator: UP, DOWN, or IDLE.
	dropoffs    *FloorSet    // Which dropoffs (destinations) are requested
	pickupsUp   *FloorSet    // Which pickups (origins) are requested UP
	pickupsDown *FloorSet    // Which pickups (origins) are requested DOWN
	chPickups   chan Pickup  // System sends us pickup demands
	chDropoffs  chan Dropoff // System (or Passenger) sends us dropoff requests from inside elevator.
	chArrivals  chan Arrival // We send when we arrive at a floor (in a direction). FUTURE: Should send dir=IDLE if no outstanding reqs.
	waiters     ArrivalListeners
	drive       *elevatorDriver
	//	chPickupQueries chan Pickup // System asks for pickup estimates
	//	chPickupQueryEstimates chan PickupEstimate // System asks for pickup estimates
}

func NewElevator(id int, numFloors int) *Elevator {
	e := &Elevator{id, numFloors, 0, 0, IDLE,
		newFloorSet(numFloors), newFloorSet(numFloors), newFloorSet(numFloors),
		make(chan Pickup), make(chan Dropoff), make(chan Arrival),
		make(ArrivalListeners), newDriver(id)}
	go e.mainLoop()
	return e
}

//func (e *Elevator) PickupQueries() chan<- Pickup { return e.chPickupQueries }
//func (e *Elevator) PickupQueryEstimates() chan<- PickupEstimate { return e.chPickupQueryEstimates }
func (e *Elevator) Id() int                  { return e.id }
func (e *Elevator) Pickups() chan<- Pickup   { return e.chPickups }
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

// Tells to Drive to go to dest, and checks response. If accepted, updates e.dir and e.dest
func (e *Elevator) gotoFloor(dest Floor) {
	if dest == e.floor || dest == e.dest { // This check may be unnecessary, better panic instead?
		log.Printf("Elevator-%d: WARNING: superfluous gotoFloor\n", e.id)
		return
	}

	// Call drive synchronously (via chan + wait for reply).
	chReply := make(chan Floor)
	log.Printf("Elevator-%d sending drive to %v", e.id, dest)
	e.drive.chRequests <- DriverDestRequest{dest, chReply}
	newDest := <-chReply
	e.dir = e.floor.DirectionTo(e.dest) // whether the new e.dest is the specified dest, or if it failed, stil udpate dir.
	if newDest == dest {
		log.Printf("Elevator-%d drive accepted new dest %v", e.id, dest)
		e.dest = dest
		e.dir = e.floor.DirectionTo(dest)
	} else {
		log.Printf("Elevator-%d drive rejected new dest %v, sticking with %v", e.id, dest, newDest)
	}
}

func (e *Elevator) mainLoop() {
	for {
		select {
		//		case pickupQuer := <-e.chPickupQueries:
		//			// Passenger outside elevator requests pickup. System requests estimates from several elevators.
		//			// Return #stops/distance/etc. before this pickup
		//			e.onPickupQuery(pickupQuery)    // FUTURE: Handle proposals.

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

	// If we are stopped at this floor, notify the pickup now.
	if e.dir == IDLE && e.floor == pickup.Floor {
		log.Printf("Elevator-%d notifying arrival on channel %v", e.id, pickup.Done)
		go func() {
			pickup.Done <- Arrival{pickup.Floor, pickup.Dir, e}
		}()
		return
	}

	e.waiters.addPickupListener(pickup)

	// If we're already aware of this pickup FloorDir, nothing to do.
	if e.pickups(pickup.Dir).set(pickup.Floor) { // set() returns previous value.
		log.Printf("Elevator-%d has this pickup already\n", e.id)
		return
	}

	// Decide whether to go to the new pickup instead.
	if e.dir == IDLE {
		// Therefore we have no other requests outstanding. FUTURE: Assert that.
		// FUTURE: This will change if delay after a stop to allow passengers to enter dropoff requests.
		e.gotoFloor(pickup.Floor)
	} else if e.dir == pickup.Dir && pickup.Floor.between(e.floor, e.dest) {
		// Pickup lies en route to our current dest, and is the same direction. Let's go there first.
		// After pickup, we will continue towards to our previous destination.
		e.gotoFloor(pickup.Floor)
	} else {
		// Either:
		// - The new pickup is en route to our current destination, but wants to go the opposite direction.
		// - The new pickup is beyond our current destination.
		// - The new pickup lies in the opposite direction as our destination.
		// In these cases, don't change our destination. We'll get to it later.
	}
}

func (e *Elevator) onDropoffReq(dropoff Dropoff) {
	log.Printf("Elevator-%d received req %v\n", e.id, dropoff)

	// If we are stopped at this floor, notify the pickup now.
	if e.dir == IDLE && e.floor == dropoff.Floor {
		log.Printf("Elevator-%d notifying arrival on channel %v", e.id, dropoff.Done)
		go func() {
			dropoff.Done <- Arrival{dropoff.Floor, IDLE, e}
		}()
		return
	}

	e.waiters.addDropoffListener(dropoff)

	if !e.dropoffs.set(dropoff.Floor) { // returns previous value
		if e.dir == IDLE {
			e.gotoFloor(dropoff.Floor)
		} else if dropoff.Floor.between(e.floor, e.dest) {
			e.gotoFloor(dropoff.Floor)
		}
	}
}

// onArrival (if s.stopping)
func (e *Elevator) onDriveNotification(s DriverStopNotification) {
	e.floor = s.floor
	if s.stopping {
		if s.floor != e.dest {
			log.Printf("Elevator-%d WARNING: got stop notification at %s, but dest = %s\n", e.id, s.floor, e.dest)
		}

		e.dropoffs.clear(e.floor)
		e.pickups(e.dir).clear(e.floor) // FUTURE: signal correct pickup light to clear.

		arrival := Arrival{e.floor, e.dir, e}
		e.waiters.notifyArrival(arrival) // Notifies all waiters

		// TODO: Passengers we just picked up have not entered their desired stop.
		// 		 We should wait for some time before choosing our next stop.
		//		 But we can't just sleep (that would block receive requests).
		//		 One approach is to return a timer which we mix into the mainLoop select{}
		dest, ok := e.calculateNextStop()
		if ok {
			// Very special case (ick): the current floor has pickup in opposite direction
			if dest == e.floor {
				e.pickups(e.dir.opposite()).clear(e.floor) // FUTURE: signal correct pickup light to clear.
				e.waiters.notifyArrival(Arrival{e.floor, e.dir.opposite(), e})
				e.dest = e.floor
				e.dir = IDLE
			} else {
				e.gotoFloor(dest) // sets e.dest, e.dir
			}
		} else {
			e.dest = e.floor
			e.dir = IDLE
		}
	}
}

// Determines the next stop, based on the current floor, dir, dest + dropoffs & pickups.
// We will continue in current direction if any dropoffs, pickups lay in that direction.
// Returns a tuple (next floor, is valid).
func (e *Elevator) calculateNextStop() (dest Floor, ok bool) {
	// TODO: What if e.dir == IDLE??
	if e.dir == IDLE {
		panic("calculateNextStop when IDLE")
	}

	floor := e.floor
	dir := e.dir
	dirOpposite := e.dir.opposite()

	// Determine next stop. In priority order, this is:

	// 1. The nearest dropoff or pickup (where pickup.dir == current direction)
	// 		which lies beyond e.floor in current direction.
	dest, ok = nearestInFloorSets(floor, dir, e.dropoffs, e.pickups(dir))
	if ok {
		return
	}

	// 2. The furthest pickup (where pickup.dir == OPPOSITE direction)
	// 		which lies AT OR beyond e.floor in current direction.
	dest, ok = e.pickups(dirOpposite).furthest(floor, dir)
	if ok {
		return
	}

	// OK, there's in our current direction. Find something in the other direction.

	// 3. The nearest dropoff or pickup (where pickup.dir == OPPOSITE direction)
	// 		which lies beyond e.floor in OPPOSITE direction
	dest, ok = nearestInFloorSets(floor, dirOpposite, e.dropoffs, e.pickups(dirOpposite)) // REVIEW: was pickups(dir)
	if ok {
		return
	}

	// 4. The furthest pickup (where pickup.dir == current direction)
	// 		which lies AT OR beyond e.floor in OPPOSITE direction
	//	  E.g., if dir == DOWN, find the highest pickup.
	dest, ok = e.pickups(dir).furthest(floor, dirOpposite) // REVIEW: was pickups(dirOpposite)
	if ok {
		return
	}

	// FUTURE: Does that cover all cases?

	return InvalidFloor, false
}

// Keeps track of those waiting for an Arrival
type ArrivalListeners map[FloorDir][]chan<- Arrival // Tracks for each FloorDir

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
			log.Printf("Elevator-%d notifying arrival on channel %v", arrival.Conveyor.Id(), ch)
			go func() {
				ch <- arrival // FUTURE: Handle closed channel
			}()
		}
		m[floorDir] = nil
	}
}
