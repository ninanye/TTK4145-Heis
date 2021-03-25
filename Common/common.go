package Common

import hw "../Driver/elevio"

// import "fmt"

const (
	NumFloors    = 4
	NumButtons   = 3
	NumElevators = 2
)

type Order struct {
	Floor  int
	Button hw.ButtonType
	Id     string
}

type ElevState int

const (
	IDLE ElevState = iota
	MOVING
	DOOROPEN
)

type Elevator struct {
	Id         string   
	Floor      int
	Dir        hw.MotorDirection //Both direction and elevator behaviour in this variable?
	State      ElevState
	Online     bool
	OrderQueue [NumFloors][NumButtons]bool // Order_queue?
	Obstructed bool
}

type HardwareChannels struct {
	HwButtons     chan hw.ButtonEvent
	HwFloor       chan int
	HwObstruction chan bool
}

type MessageType int

const (
	ORDER MessageType = iota
	ELEVSTATUS
	CONFIRMATION
)

type Message struct {
	OrderMsg    Order
	ElevatorMsg Elevator
	MsgType     MessageType
	MessageId   int
	ElevatorId  string
}

type NetworkChannels struct {
	//PeerUpdateCh chan peers.PeerUpdate
	PeerTxEnable   chan bool
	BcastMessage   chan Message
	RecieveMessage chan Message
}

type OrderChannels struct {
	//From assigner to distributer
	SendOrder chan Order
	//From distributer to assigner
	OrderBackupUpdate chan Order
	RecieveElevUpdate chan Elevator
	//From distributor to executer
	LocalOrder chan Order
	//From executer to distributor
	LocalElevUpdate chan Elevator
}
