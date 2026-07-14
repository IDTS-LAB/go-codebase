package events

var registry = map[string]func() interface{}{}

func Register(eventType string, factory func() interface{}) {
	registry[eventType] = factory
}

func CreatePayload(eventType string) interface{} {
	factory, ok := registry[eventType]
	if !ok {
		return nil
	}
	return factory()
}
