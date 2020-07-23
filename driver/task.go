package driver

type Service struct {
	Description string
	Name        string
	Namespace   string
}

type Task struct {
	Description  string
	Name         string
	Providers    []map[string]interface{}
	ProviderInfo map[string]interface{}
	Services     []Service
	Source       string
	Version      string
}
