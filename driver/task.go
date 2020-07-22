package driver

type Service struct {
	Datacenter  string
	Description string
	Name        string
	Namespace   string
	Tag         string
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
