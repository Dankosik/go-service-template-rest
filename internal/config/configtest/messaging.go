package configtest

const (
	tlsBrokerURL = "tls://broker.example:4222"
	eventsStream = "EVENTS"
)

// MessagingCase is one table-only input shared by composition roots that must
// prove their typed NATS mapping against internal/config's accepted values.
type MessagingCase struct {
	Name          string
	URLs          string
	Stream        string
	Plaintext     bool
	ConfigRejects bool
}

func MessagingCases() []MessagingCase {
	return []MessagingCase{
		{Name: "valid plaintext", URLs: "nats://broker.example:4222", Stream: eventsStream, Plaintext: true},
		{Name: "valid tls", URLs: tlsBrokerURL, Stream: eventsStream},
		{Name: "valid websocket", URLs: "ws://broker.example:4222", Stream: eventsStream, Plaintext: true},
		{Name: "valid secure websocket", URLs: "wss://broker.example:4222", Stream: eventsStream},
		{Name: "two urls", URLs: "tls://a.example:4222,tls://b.example:4222", Stream: eventsStream},
		{Name: "userinfo", URLs: "tls://user@broker.example:4222", Stream: eventsStream, ConfigRejects: true},
		{Name: "unsupported scheme", URLs: "amqp://broker.example:5672", Stream: eventsStream, ConfigRejects: true},
		{Name: "plaintext without opt-in", URLs: "nats://broker.example:4222", Stream: eventsStream, ConfigRejects: true},
		{Name: "no host", URLs: "tls://", Stream: eventsStream, ConfigRejects: true},
		{Name: "empty urls", URLs: "", Stream: eventsStream},
		{Name: "empty stream", URLs: tlsBrokerURL, Stream: "", ConfigRejects: true},
		{Name: "stream with separator", URLs: tlsBrokerURL, Stream: "EV.ENTS", ConfigRejects: true},
		{Name: "stream with wildcard", URLs: tlsBrokerURL, Stream: "EV*NTS", ConfigRejects: true},
		{Name: "stream with full wildcard", URLs: tlsBrokerURL, Stream: "EVENTS>", ConfigRejects: true},
		{Name: "stream with space", URLs: tlsBrokerURL, Stream: "EV ENTS", ConfigRejects: true},
		{Name: "stream with path", URLs: tlsBrokerURL, Stream: "EV/ENTS", ConfigRejects: true},
		{Name: "stream with tab", URLs: tlsBrokerURL, Stream: "EV\tENTS", ConfigRejects: true},
	}
}
