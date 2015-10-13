package henchman

type Machine struct {
	Hostname  string
	Group     string
	Transport TransportInterface
}
