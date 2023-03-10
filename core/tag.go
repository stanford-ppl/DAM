package core

type TagType[D any, U any] struct {
	TagID int
}

type InputTagUpdater[D any, U any] interface {
	CanRun(update U) bool
	Update(state D, update U, enabled bool) D
}

type InputTag[D any, U any] struct {
	Tag       TagType[D, U] // Points back to the Tag
	InputPort Port

	State D

	Null    U
	Updater InputTagUpdater[D, U]
}

type OutputTagPublisher[D any, U any] interface {
	// Writes a value of U to OutputChannel
	// Can depend on the current state of the node
	Publish(state interface{}) U
	HasPublish(state interface{}) bool
}

type OutputTag[D any, U any] struct {
	Tag        TagType[D, U]
	OutputPort Port
	Publisher  OutputTagPublisher[D, U]
}
