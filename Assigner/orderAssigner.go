package Assigner

import (
	"fmt"

	. "../Common"

	hw "../Driver/elevio"

	localip "../Network/network/localip"

	exe "../Executer"

	"os"
)

var AllElevators map[string]Elevator
var OrderBackup map[string][]Order
var SetLights map[string]bool

func GetElevIP() string {
	// Adds elevator-ID (localIP + process ID)
	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println(err)
		localIP = "DISCONNECTED"
	}
	id := fmt.Sprintf("%s-%d", localIP, os.Getpid())
	return id
}

func AssignOrder(hwChan HardwareChannels, orderChan OrderChannels) {
	for {
		select {
		case buttonPress := <-hwChan.HwButtons:
			id := "No ID"
			// Cost function returning ID of elevator taking the order
			if buttonPress.Button == hw.BT_Cab {
				id = GetElevIP()
			} else {
				id = costFunction(AllElevators)
			}
			if !duplicateOrder(buttonPress.Button, buttonPress.Floor) {
				newOrder := Order{Floor: buttonPress.Floor, Button: buttonPress.Button, Id: id}
				fmt.Println("Sending normal order")
				orderChan.SendOrder <- newOrder
			}

		case offlineElev := <-orderChan.ReassignOrders:
			id := "No ID"
			fmt.Println("Recieved lost peer")
			elev := AllElevators[offlineElev]
			fmt.Println("Elevator 1", elev)
			for floor := 0; floor < NumFloors; floor++ {
				for btn := 0; btn < NumButtons-1; btn++ {
					if elev.OrderQueue[floor][btn] {
						fmt.Println("Going into if statement")
						id = costFunction(AllElevators)
						elev.OrderQueue[floor][btn] = false
						AllElevators[offlineElev] = elev
						fmt.Println("Elevator 1", elev)
						if !duplicateOrder(hw.ButtonType(btn), floor) {
							newOrder := Order{Floor: floor, Button: hw.ButtonType(btn), Id: id}
							fmt.Println("Sending reassigned order!")
							orderChan.SendOrder <- newOrder
						}

					}
				}
			}
		}
	}
}

func UpdateAssigner(orderChan OrderChannels, netChan NetworkChannels) {
	for {
		select {
		case newOrder := <-orderChan.OrderBackupUpdate:
			OrderBackup[newOrder.Id] = append(OrderBackup[newOrder.Id], newOrder)
			// Make function that deletes orders from backup when finished
		case updatedElev := <-orderChan.RecieveElevUpdate:
			AllElevators[updatedElev.Id] = updatedElev
			setAllLights()
			if !updatedElev.Online && updatedElev.Id == GetElevIP() {
				netChan.PeerTxEnable <- false
			} else {
				netChan.PeerTxEnable <- true
			}
		}
	}
}

func PeerUpdate(netChan NetworkChannels, orderChan OrderChannels) {
	for {
		select {
		case p := <-netChan.PeerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)
			if p.New != "" {
				if elev, foundPeer := AllElevators[p.New]; foundPeer { // If elevator is found again, going online
					elev.Online = true
					AllElevators[p.New] = elev
					//When engine p

				} else { // If elevator is new, needs to be created
					elev := Elevator{
						Id:         p.New,
						Floor:      0,
						Dir:        hw.MD_Stop,
						State:      IDLE,
						Online:     true,
						OrderQueue: [NumFloors][NumButtons]bool{},
						Obstructed: false,
					}
					AllElevators[p.New] = elev
				}
				NumElevators++
				//orderChan.CopyOfAllElevators <- AllElevators
			}
			if len(p.Lost) > 0 {
				for _, lostPeer := range p.Lost { // If elevator is lost, going offline
					fmt.Println("Elevator going offline")
					elev := AllElevators[lostPeer]
					elev.Online = false
					AllElevators[lostPeer] = elev
					NumElevators--
					orderChan.ReassignOrders <- lostPeer
				}
				// orderChan.CopyOfAllElevators <- AllElevators

			}
		}
	}
}

func costFunction(allElev map[string]Elevator) string {
	minTime := -1
	minId := ""
	for id, elev := range allElev {
		if elev.Online {
			time := timeToIdle(elev)
			if time < minTime || minTime == -1 {
				minTime = time
				minId = id
			}
		}
	}
	return minId

}

func timeToIdle(elev Elevator) int {
	/*e := elev
	e.OrderQueue[floor][btn] = true

	arrivedAtRequest := false

	ifEqual := func(inner_b hw.ButtonType, inner_f int) {
		if inner_b == btn && inner_f == floor {
			arrivedAtRequest = true
		}
	}*/

	duration := 0

	switch elev.State {
	case IDLE:
		elev.Dir = exe.ChooseDirection(elev, elev.Dir)
		if elev.Dir == hw.MD_Stop {
			return duration
		}
		break
	case MOVING:
		duration += TravelTime / 2 //Define travel time later
		elev.Floor += int(elev.Dir)
		break
	case DOOROPEN:
		duration -= DoorOpenTime / 2
	}
	for {
		if exe.ShouldStop(elev) {
			elev = exe.ClearOrdersAtCurrentFloor(elev)
			if elev.Dir == hw.MD_Stop {
				return duration
			}
			duration += DoorOpenTime
			elev.Dir = exe.ChooseDirection(elev, elev.Dir)

		}
		elev.Floor += int(elev.Dir)
		duration += TravelTime
	}
}

func setAllLights() {
	ID := GetElevIP()
	var lightsOff bool
	for floor := 0; floor < NumFloors; floor++ {
		for btn := 0; btn < NumButtons; btn++ {
			for id, elev := range AllElevators {
				SetLights[id] = false
				if btn == hw.BT_Cab && id != ID {
					continue
				}
				if elev.OrderQueue[floor][btn] && elev.Online {
					SetLights[id] = true
					hw.SetButtonLamp(hw.ButtonType(btn), floor, true)
				}
			}
			lightsOff = true
			for _, val := range SetLights {
				if val == true {
					lightsOff = false
				}
			}
			if lightsOff {
				hw.SetButtonLamp(hw.ButtonType(btn), floor, false)
			}
		}
	}
}

func duplicateOrder(btn hw.ButtonType, floor int) bool {
	ID := GetElevIP()
	for id, elev := range AllElevators {
		if btn == hw.BT_Cab && id != ID {
			continue
		}
		if elev.OrderQueue[floor][btn] {
			fmt.Println("Returning TRUE to duplicate")
			return true
		}
	}
	fmt.Println("Returning FALSE to duplicate")
	return false
}
