package manager

//go:generate mockery --name=Health --filename=health.go --output=../mocks/managers

const (
	HealthSubsystemName = "health"
)

type Health interface {
	Check() error
}

type BasicHealth struct{}

func (h *BasicHealth) Check() error {
	return nil
}
