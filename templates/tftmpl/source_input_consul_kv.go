package tftmpl

var (
	_ SourceInput = (*ConsulKVSourceInput)(nil)
)

// ConsulKVSourceInput handles appending a run source variable's relevant templating for Terraform
// generated files
type ConsulKVSourceInput struct {
	ConsulKVMonitor
}
