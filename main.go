package main

import (
	"fmt"
	"sync"
	"time"
)

type ChamberState int

const (
	Pressurized ChamberState = iota
	Empty
	Changing
)

func (s ChamberState) String() string {
	switch s {
	case Pressurized:
		return "pressurized"
	case Empty:
		return "empty"
	case Changing:
		return "changing"
	default:
		return "unknown"
	}
}

type Airlock struct {
	// One mutex + one condition variable keep all state transitions atomic and coordinated.
	mu sync.Mutex
	cv *sync.Cond

	outsideDoorOpen bool
	insideDoorOpen  bool
	chamber         ChamberState
	occupied        bool // True while someone is physically in the chamber.

	// Number of goroutines currently waiting to enter from each side.
	// Used to avoid closing a door right before a waiting astronaut can use it.
	waitingInside   int
	waitingOutside  int
	verbose         bool
}

type DoorSide int

const (
	Inside DoorSide = iota
	Outside
)

func NewAirlock() *Airlock {
	return NewAirlockWithLogging(true)
}

func NewAirlockWithLogging(verbose bool) *Airlock {
	a := &Airlock{
		chamber: Pressurized,
		verbose: verbose,
	}
	a.cv = sync.NewCond(&a.mu)
	return a
}

func (a *Airlock) log(actor, action string) {
	if !a.verbose {
		return
	}
	fmt.Printf("%-18s | %-22s | outside=%v inside=%v chamber=%s occupied=%v\n",
		actor, action, a.outsideDoorOpen, a.insideDoorOpen, a.chamber, a.occupied)
}

func (a *Airlock) assertSafe() {
	// Central safety invariants: if any of these fail, the simulation is in an impossible/unsafe state.
	if a.outsideDoorOpen && a.insideDoorOpen {
		panic("catastrophic failure: both doors are open")
	}
	if a.outsideDoorOpen && a.chamber != Empty {
		panic("invalid state: outside door open while chamber is not empty")
	}
	if a.insideDoorOpen && a.chamber != Pressurized {
		panic("invalid state: inside door open while chamber is not pressurized")
	}
}

func (a *Airlock) openDoor(actor string, side DoorSide) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if side == Inside {
		// Inner door can only open while the chamber is pressurized and outer door is closed.
		for a.outsideDoorOpen || a.chamber != Pressurized {
			a.cv.Wait()
		}
		// Idempotent open: if the door is already open in a valid state, do nothing.
		if !a.insideDoorOpen {
			a.insideDoorOpen = true
			a.log(actor, "open inside door")
		}
	} else {
		// Outer door can only open while the chamber is empty and inner door is closed
		for a.insideDoorOpen || a.chamber != Empty {
			a.cv.Wait()
		}
		// Idempotent open avoids deadlocks when both actors race to "open" the same side
		if !a.outsideDoorOpen {
			a.outsideDoorOpen = true
			a.log(actor, "open outside door")
		}
	}
	a.assertSafe()
	a.cv.Broadcast()
}

func (a *Airlock) closeDoor(actor string, side DoorSide) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if side == Inside {
		// If someone is already waiting on this side and chamber is free, keep the door available
		if !a.occupied && a.waitingInside > 0 {
			a.cv.Broadcast()
			return
		}
		a.insideDoorOpen = false
		a.log(actor, "close inside door")
	} else {
		// Same idea for the outside side
		if !a.occupied && a.waitingOutside > 0 {
			a.cv.Broadcast()
			return
		}
		a.outsideDoorOpen = false
		a.log(actor, "close outside door")
	}
	a.assertSafe()
	a.cv.Broadcast()
}

func (a *Airlock) enterFromInside(actor string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.waitingInside++
	defer func() {
		a.waitingInside--
	}()
	for !a.insideDoorOpen || a.occupied {
		a.cv.Wait()
	}
	a.occupied = true
	a.log(actor, "enter chamber (inside)")
	a.cv.Broadcast()
}

func (a *Airlock) exitToOutside(actor string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for !a.outsideDoorOpen || !a.occupied {
		a.cv.Wait()
	}
	a.occupied = false
	a.log(actor, "exit chamber (outside)")
	a.cv.Broadcast()
}

func (a *Airlock) enterFromOutside(actor string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.waitingOutside++
	defer func() {
		a.waitingOutside--
	}()
	for !a.outsideDoorOpen || a.occupied {
		a.cv.Wait()
	}
	a.occupied = true
	a.log(actor, "enter chamber (outside)")
	a.cv.Broadcast()
}

func (a *Airlock) exitToInside(actor string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for !a.insideDoorOpen || !a.occupied {
		a.cv.Wait()
	}
	a.occupied = false
	a.log(actor, "exit chamber (inside)")
	a.cv.Broadcast()
}

func (a *Airlock) transitionChamber(actor string, from, to ChamberState, startAction, finishAction string) {
	a.mu.Lock()
	// Pressurization changes are only allowed with both doors closed.
	for a.insideDoorOpen || a.outsideDoorOpen || a.chamber != from {
		a.cv.Wait()
	}
	a.chamber = Changing
	a.log(actor, startAction)
	a.mu.Unlock()

	// Simulate that pressure changes take time.
	time.Sleep(80 * time.Millisecond)

	a.mu.Lock()
	a.chamber = to
	a.assertSafe()
	a.log(actor, finishAction)
	a.cv.Broadcast()
	a.mu.Unlock()
}

func (a *Airlock) depressurize(actor string) {
	a.transitionChamber(actor, Pressurized, Empty, "start depressurizing", "finish depressurizing")
}

func (a *Airlock) pressurize(actor string) {
	a.transitionChamber(actor, Empty, Pressurized, "start pressurizing", "finish pressurizing")
}

func insideAstronaut(a *Airlock, wg *sync.WaitGroup) {
	defer wg.Done()
	// Scipione starts inside the ISS and wants to go outside
	actor := "Scipione (inside)"

	a.openDoor(actor, Inside)
	a.enterFromInside(actor)
	a.closeDoor(actor, Inside)
	a.depressurize(actor)
	a.openDoor(actor, Outside)
	a.exitToOutside(actor)
	a.closeDoor(actor, Outside)
}

func outsideAstronaut(a *Airlock, wg *sync.WaitGroup) {
	defer wg.Done()
	// Anibal starts outside the ISS and wants to come back in
	actor := "Anibal (outside)"

	a.openDoor(actor, Outside)
	a.enterFromOutside(actor)
	a.closeDoor(actor, Outside)
	a.pressurize(actor)
	a.openDoor(actor, Inside)
	a.exitToInside(actor)
	a.closeDoor(actor, Inside)
}

func main() {
	a := NewAirlock()
	var wg sync.WaitGroup

	wg.Add(2)
	go insideAstronaut(a, &wg)
	go outsideAstronaut(a, &wg)
	wg.Wait()

	a.mu.Lock()
	a.log("system", "completed safely")
	a.mu.Unlock()
}
