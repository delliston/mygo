package elevator

import "fmt"

/* Assumptions:
		Pickup specifies pickup floor + direction.
		Dropoff specifies dropoff floor.
		Dropoffs and pickups cannot be cancelled.

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
			Go towards dropoff
			While moving, service any dropoff or DOWN pickups.
			Do not service any upward dropoffs or pickups.
		on pickup: (e.g., above and DOWNward)
			doTopDownwardPickup() -- defined above.

	If going DOWN, reverse the UP clause.

	CASE: An IDLE elevator at floor 1 receives a pickup DOWN from floor 10. It heads toward 10.
		  If it gets an UP request from floor 15, en route, it will not service it before taking 10 DOWN.
		  If pickup DOWN @ 10 is serviced, and passenger requests dropoff above, what should we do?

-------

	USE CASES
	- A pickup is issued to an elevator A. Later, elevator B stops at that floor, going in the right direction.
	  Then the first pickup should be cancelled.
	- How to prevent starvation? Should pickup requests be handled in the order received?

	THINKING
	- Should elevator be able to operate independently of the system?
	- Whats the simplest approach?
	  - @STEP
	  	if direction != 0:
		  	floor += direction
		  	if floor in pickups or dropoffs, open door, delete pickup & dropoff
		  	if no pickups/dropoffs:
		  		if pickups or dropoffs:
		  			direction *= 1
		  		else:
		  			direction = 0
	  - @DROPOFF-requested
	  	if direction == 0:
	  - @PICKUP-proposal
	  		return estimate of how many steps until pickup possible

	    add to dropoffs

	 - Yes, elevator receives a pickup request and acks it with estimated time
*/

// Elevator listens to commands on a channel:
//		dropoff [floorNum] (request issued inside elevator by passenger)
//		pickup [floorNum] (request issued outside elevator by potential passenger)
//	It is not possible to cancel a dropoff request (as in real elevators).
type Elevator struct {
	id int
	numFloors int
	floor Floor  	// The last floor we passed, or (if dir==IDLE, the floor we are sitting on). The floor has already been serviced.
	dir Direction   // Current direction of the elevator: UP, DOWN, or IDLE.
	dest Floor		// The current destination. dir == floor.directionTo(dest). If dir == IDLE, dest == floor.
	pickup *Pickup   // Special: The pickup we are rushing towards; after arriving we will set dir = pickup.dir
	dropoffs *FloorSet    // Which dropoffs (destinations) are requested
	pickupsUp *FloorSet   // Which pickups (origins) are requested UP
	pickupsDown *FloorSet   // Which pickups (origins) are requested DOWN
//	chPickupQueries chan Pickup // System asks for pickup estimates
//	chPickupQueryEstimates chan PickupEstimate // System asks for pickup estimates
	chPickups chan Pickup // System sends us pickup demands
	chPickupsDone chan Pickup // System sends us pickup demands
	chDropoffs chan Dropoff // System (or Passenger) sends us dropoff requests from inside elevator.
	drive *elevatorDriver
}
func NewElevator(id int, numFloors int, chPickupQueryEstimates chan PickupEstimate, chPickupsDone chan Pickup) (*Elevator) {
	return &Elevator{ id, numFloors, 0, IDLE, 0, nil, newFloorSet(numFloors), newFloorSet(numFloors), newFloorSet(numFloors),
					  make(chan Pickup), chPickupQueryEstimates, make(chan Pickup), chPickupsDone, newDriver() }
}
func (e *Elevator) PickupQueries() chan<- Pickup { return e.chPickupQueries }
func (e *Elevator) PickupQueryEstimates() chan<- PickupEstimate { return e.chPickupQueryEstimates }
func (e *Elevator) Pickups() chan<- Pickup { return e.chPickups }
func (e *Elevator) PickupsDone() chan<- Pickup { return e.chPickupsDone }

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
	e.dest = pickup.floor
	e.dir = e.floor.directionTo(pickup.floor)
	e.sendDriveTo(pickup.floor) // updates e.dir, e.nextStop
}

func (e *Elevator) gotoFloor(floor Floor) {
	e.pickup = nil
	e.dest = floor
	e.dir = e.floor.directionTo(floor)
	e.sendDriveTo(floor) // updates e.dir, e.nextStop
}

func (e *Elevator) mainLoop() {
	for {
		select {
		case pickup := <-e.chPickupQueries:
			// Passenger outside elevator requests pickup System Request for pickup proposal.
			// Return #stops/distance before this pickup
			e.onPickupQuery(pickup)    // FUTURE: Handle proposals.

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

func (e *Elevator) onPickupQuery(pickup Pickup) {
	stopsUntilPickup := 0
	distanceUntilPickup := 0
	goinThereAnyway := false
	// TODO: Fill in these values
	e.chPickupQueryEstimates <- &PickupEstimate{ pickup, stopsUntilPickup, distanceUntilPickup, goinThereAnyway }
}

func (e *Elevator) onPickupReq(pickup Pickup) {
	if !e.pickups(pickup.dir).set(pickup.floor) {    // returns previous value. If it wasn't set before, then:
		// Decide whether to go to the new pickup instead.
		if e.dir == IDLE {
			// 1. We are IDLE
			e.gotoPickup(pickup)
		} else if e.pickup != nil {
			// 2. We are rushing (in direction e.dir) to a pickup AND either:
			//    A) The new pickup lies between e.floor and e.pickup.floor
			//	     AND e.dir == pickup.dir   // after pickup, we continue beyond it
			//    B) The new pickup lies beyond e.pickup.floor (in direction e.dir) from e.floor
			//	     AND e.dir == opposite of (pickup.dir) // after pickup, we switch direction
			if pickup.floor.between(e.floor, e.pickup.floor) {
				if e.dir == pickup.dir {
					e.gotoPickup(pickup)
				}
			} else {
				if e.dir == pickup.dir.opposite() {
					e.gotoPickup(pickup)
				}
			}
		} else {
			// 3. This pickup lies en route to and in the direction of to our current destination.
			if e.dir == pickup.dir && pickup.floor.between(e.floor, e.dest) {
				e.gotoFloor(pickup.floor)
			}
		}
	}
}

func (e *Elevator) onDropoffReq(dropoff Dropoff) {
	if !e.dropoffs.set(dropoff.floor) {    // returns previous value
		if e.dir == IDLE {
			e.gotoFloor(dropoff.floor)
		} else if e.pickup != nil {
			// We ignore the dropoff request. We are rushing to a pickup.
			// Ouch, that's bad if the pickup comes when user is entering elevator and selecting a floor.
			// FUTURE: For some period (doors open), block pickup requests, only accept dropoff requests.
		} else if dropoff.floor.between(e.floor, e.dest) {
			e.gotoFloor(dropoff.floor)
		}
	}
}

func (e *Elevator) onDriveNotification(s DriverStopNotification) {
	e.floor = s.floor
	if s.stopping {
		if e.pickup != nil && e.pickup.floor == e.floor {
			// We've hit our target pickup. We may switch direction.
			e.dir = e.pickup.dir
			e.pickup = nil
		}
		e.pickups(e.dir).clear(e.floor)        // FUTURE: signal correct pickup light to clear.
		e.chPickupsDone <- &Pickup{ e.floor, e.dir }	// FUTURE: Send notification to each pickup we received for this floor.
		e.dropoffs.clear(e.floor)
		// TODO: Signal passenger that pickup or dropoff occurred. Could return a timer to wait on before proceeding.
		// Should wait for some time for this to arrive before specifying next

		// Determine next stop
		dest := findNext(e.floor, e.dir, e.dropoffs, e.pickups(e.dir)) // returns first arg if no place to go
		if dest == e.floor {
			e.dir = IDLE
		} else {
			e.gotoFloor(dest)
		}
	}
}


// Sends cmd to Drive to set floor, and checks response. If accepted, updates e.dir and e.nextStop
func (e *Elevator) sendDriveTo(dest Floor) {
	chReply := make(chan Floor)
	e.chDrive<- DriverDestRequest{dest, chReply}
	e.dest := <-chReply
	e.dir = e.floor.directionTo(e.dest)	// whether the new e.dest is the specified dest, or if it failed, stil udpate dir.
}

// Find the next Floor (in the direction), among the specified FloorSets.
// If none, return cur.
func findNext(cur Floor, direction Direction, floorSets... FloorSet) Floor {
	next := cur
	for fs := range floorSets {
		fsNext := fs.next(cur, direction)
		if fsNext != cur && fsNext < next {
			next = fsNext
		}
	}
	return next
}



// elevatorDriver receives requests to stop at a particular floor.It knows its direction and current destination floor.
// If it can stop at the new floor, it sets it as new destination and returns this value.
// If it cannot (because it has passed the floor, or is near the floor and moving too fast to stop),
// then it does not change its destination, and returns the old value.
type elevatorDriver struct {
	chRequests chan DriverDestRequest
	chNotifications chan DriverStopNotification
}
func newDriver() *elevatorDriver {
	return &elevatorDriver{ make(chan DriverDestRequest), make(chan DriverStopNotification) }
}

type DriverDestRequest struct {
	floor Floor
	chReply <-chan Floor
}

type DriverStopNotification struct {
	floor Floor
	stopping bool
}

func (d *elevatorDriver) mainLoop(floor Floor, chStopNotifications chan<- Floor, chStopAt <-chan DriverDestRequest) {
	dir := IDLE
	dest := floor
	for {
		timer := nil
		select {
		case s := <-chStopAt:
			if dir == IDLE {
				dest = s.floor    // FUTURE: set speed
			} else if s.floor.between(floor, dest) {   // FUTURE: check also if we slow down in time
				dest = s.floor    // FUTURE: may reduce speed.
			} else if floor.directionTo(s.Floor) == dir {
				dest = s.floor    // FUTURE: may increase speed
			}
			s.chReply<- dest

		case <-timer:    // Passing or stopping at a floor.
			floor = floor + dir
			d.chNotifications<- &DriverStopNotification{floor, floor == dest}
		}

		// Update dir, timer whether we continue or stop.
		if floor == dest {
			dir = IDLE
			timer = nil
		} else {
			dir = floor.directionTo(dest)
			timer = time.after(1 * time.Second)
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
