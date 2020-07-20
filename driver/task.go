package driver

type Service struct {
	Description string
	Name        string
	Namespace   string
}

type Task struct {
	Description string
	Name        string
	// Providers []Provider
	Services []Service
	Source   string
	Version  string
}
