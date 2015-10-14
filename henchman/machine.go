package henchman

type Machine struct {
	Hostname  string
	Vars      VarsMap
	Transport TransportInterface
}
