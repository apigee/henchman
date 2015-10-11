package henchman

type Machine struct {
	Vars      VarsMap
	Hostname  string
	Transport TransportInterface
}
