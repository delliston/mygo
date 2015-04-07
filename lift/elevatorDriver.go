package lift

import (
//	"fmt"
	"time"
	"log"
	"os"
)

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
	chRequests chan DriverDestRequest			// We receive requests here
	chNotifications chan DriverStopNotification	// We send notifications here
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
				log.Printf("Elevator-%d going %s changed destination from %s to %s\n", d.id, d.dir, d.dest, req.floor)
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

	log.Println("All passengers have been serviced")
	os.Exit(0)
}
