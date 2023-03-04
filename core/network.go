package core

struct CommunicationChannel[T] {
	Name string
	channel chan T
}

func NewCommunicationChannel[T](name string, capacity int) *CommunicationChannel {
	&NewCommunicationChannel[T]{Name:= name, make(chan T, capacity)}
}
