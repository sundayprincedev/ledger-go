package jobs

type AckStatus string

const (
	AckCompleted        AckStatus = "completed"
	AckAlreadyCompleted AckStatus = "already_completed"
)

type Ack struct {
	JobID  string
	Status AckStatus
}

func newAck(jobID string, status AckStatus) Ack {
	return Ack{JobID: jobID, Status: status}
}
