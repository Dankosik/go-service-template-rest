package postgresoutbox

import (
	"bytes"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const jobKind = "publish_domain_event"

// PublishJob is the immutable River record consumed by the NATS outbox worker.
type PublishJob struct {
	ID         string    `json:"id" river:"unique"`
	Type       string    `json:"type"`
	Version    uint16    `json:"version"`
	OccurredAt time.Time `json:"occurred_at"`
	Payload    []byte    `json:"payload"`
	Subject    string    `json:"subject"`
}

func (PublishJob) Kind() string {
	return jobKind
}

func (PublishJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: Queue,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: rivertype.JobStates(),
		},
	}
}

// samePublication reports whether two jobs carry the same publication data.
func (j PublishJob) samePublication(other PublishJob) bool {
	return j.ID == other.ID &&
		j.Type == other.Type &&
		j.Version == other.Version &&
		j.OccurredAt.Equal(other.OccurredAt) &&
		j.Subject == other.Subject &&
		bytes.Equal(j.Payload, other.Payload)
}
