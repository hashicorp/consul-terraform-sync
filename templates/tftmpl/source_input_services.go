package tftmpl

var (
	_ SourceInput = (*ServicesSourceInput)(nil)
)

// SourceInput handles appending a run source input's relevant templating for Terraform
// generated files
type SourceInput interface {
	Monitor
}

// ServicesSourceInput handles appending a run source variable's relevant templating for Terraform
// generated files
type ServicesSourceInput struct {
	ServicesMonitor
}
