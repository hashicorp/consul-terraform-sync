// Package oapigen provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package oapigen

import (
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Defines values for CreateTaskParamsRun.
const (
	Inspect CreateTaskParamsRun = "inspect"
	Now     CreateTaskParamsRun = "now"
)

// BufferPeriod The buffer period for triggering task execution.
type BufferPeriod struct {
	// Enabled Whether the buffer period is enabled or disabled. Defaults to the global buffer period configured for CTS.
	Enabled *bool `json:"enabled,omitempty"`

	// Max The maximum period of time to wait after changes are detected before triggering the task. Defaults to the global buffer period configured for CTS.
	Max *string `json:"max,omitempty"`

	// Min The minimum period of time to wait after changes are detected before triggering the task. Defaults to the global buffer period configured for CTS.
	Min *string `json:"min,omitempty"`
}

// CatalogServicesCondition defines model for CatalogServicesCondition.
type CatalogServicesCondition struct {
	Datacenter       *string            `json:"datacenter,omitempty"`
	Namespace        *string            `json:"namespace,omitempty"`
	NodeMeta         *map[string]string `json:"node_meta,omitempty"`
	Regexp           string             `json:"regexp"`
	UseAsModuleInput *bool              `json:"use_as_module_input,omitempty"`
}

// ClusterStatusResponse defines model for ClusterStatusResponse.
type ClusterStatusResponse struct {
	// ClusterName the name of the CTS cluster
	ClusterName string `json:"cluster_name"`

	// Members the list of CTS instances which are part of this cluster
	Members   []Member  `json:"members"`
	RequestId RequestID `json:"request_id"`
}

// Condition The condition on which to trigger the task to execute. If the task has the deprecated services field configured as a module input, it is represented here as condition.services.
type Condition struct {
	CatalogServices *CatalogServicesCondition `json:"catalog_services,omitempty"`
	ConsulKv        *ConsulKVCondition        `json:"consul_kv,omitempty"`
	Schedule        *ScheduleCondition        `json:"schedule,omitempty"`
	Services        *ServicesCondition        `json:"services,omitempty"`
}

// ConsulKVCondition defines model for ConsulKVCondition.
type ConsulKVCondition struct {
	Datacenter       *string `json:"datacenter,omitempty"`
	Namespace        *string `json:"namespace,omitempty"`
	Path             string  `json:"path"`
	Recurse          *bool   `json:"recurse,omitempty"`
	UseAsModuleInput *bool   `json:"use_as_module_input,omitempty"`
}

// ConsulKVModuleInput defines model for ConsulKVModuleInput.
type ConsulKVModuleInput struct {
	Datacenter *string `json:"datacenter,omitempty"`
	Namespace  *string `json:"namespace,omitempty"`
	Path       string  `json:"path"`
	Recurse    *bool   `json:"recurse,omitempty"`
}

// Error defines model for Error.
type Error struct {
	Message string `json:"message"`
}

// ErrorResponse defines model for ErrorResponse.
type ErrorResponse struct {
	Error     Error     `json:"error"`
	RequestId RequestID `json:"request_id"`
}

// HealthCheckResponse defines model for HealthCheckResponse.
type HealthCheckResponse struct {
	Error *Error `json:"error,omitempty"`
}

// Member defines model for Member.
type Member struct {
	// Address the address if this instance is the CTS leader, empty otherwise
	Address *string `json:"address,omitempty"`

	// Healthy the health status of the cts instance
	Healthy bool `json:"healthy"`

	// Id the id of the CTS instance
	Id string `json:"id"`

	// Leader boolean true if CTS instance is a cluster leader, false otherwise
	Leader bool `json:"leader"`

	// ServiceName the service name of the CTS instance
	ServiceName string `json:"service_name"`
}

// ModuleInput The additional module input(s) that the tasks provides to the Terraform module on execution. If the task has the deprecated services field configured as a module input, it is represented here as module_input.services.
type ModuleInput struct {
	ConsulKv *ConsulKVModuleInput `json:"consul_kv,omitempty"`
	Services *ServicesModuleInput `json:"services,omitempty"`
}

// RequestID defines model for RequestID.
type RequestID = openapi_types.UUID

// Run defines model for Run.
type Run struct {
	// ChangesPresent Whether or not infrastructure changes were detected during task inspection.
	ChangesPresent *bool   `json:"changes_present,omitempty"`
	Plan           *string `json:"plan,omitempty"`

	// TfcRunUrl Enterprise only. URL of Terraform Cloud run that corresponds to the task run.
	TfcRunUrl *string `json:"tfc_run_url,omitempty"`
}

// ScheduleCondition defines model for ScheduleCondition.
type ScheduleCondition struct {
	Cron string `json:"cron"`
}

// ServicesCondition defines model for ServicesCondition.
type ServicesCondition struct {
	CtsUserDefinedMeta *map[string]*string `json:"cts_user_defined_meta,omitempty"`
	Datacenter         *string             `json:"datacenter,omitempty"`
	Filter             *string             `json:"filter,omitempty"`
	Names              *[]string           `json:"names,omitempty"`
	Namespace          *string             `json:"namespace,omitempty"`
	Regexp             *string             `json:"regexp,omitempty"`
	UseAsModuleInput   *bool               `json:"use_as_module_input,omitempty"`
}

// ServicesModuleInput defines model for ServicesModuleInput.
type ServicesModuleInput struct {
	CtsUserDefinedMeta *map[string]string `json:"cts_user_defined_meta,omitempty"`
	Datacenter         *string            `json:"datacenter,omitempty"`
	Filter             *string            `json:"filter,omitempty"`
	Names              *[]string          `json:"names,omitempty"`
	Namespace          *string            `json:"namespace,omitempty"`
	Regexp             *string            `json:"regexp,omitempty"`
}

// Task defines model for Task.
type Task struct {
	// BufferPeriod The buffer period for triggering task execution.
	BufferPeriod *BufferPeriod `json:"buffer_period,omitempty"`

	// Condition The condition on which to trigger the task to execute. If the task has the deprecated services field configured as a module input, it is represented here as condition.services.
	Condition Condition `json:"condition"`

	// Description The human readable text to describe the task.
	Description *string `json:"description,omitempty"`

	// Enabled Whether the task is enabled or disabled from executing.
	Enabled *bool `json:"enabled,omitempty"`

	// Module The location of the Terraform module.
	Module string `json:"module"`

	// ModuleInput The additional module input(s) that the tasks provides to the Terraform module on execution. If the task has the deprecated services field configured as a module input, it is represented here as module_input.services.
	ModuleInput *ModuleInput `json:"module_input,omitempty"`

	// Name The unique name of the task.
	Name string `json:"name"`

	// Providers The list of provider names that the task's module uses.
	Providers *[]string `json:"providers,omitempty"`

	// TerraformCloudWorkspace Enterprise only. Configuration values to use for the Terraform Cloud workspace associated with the task. This is only available when used with the Terraform Cloud driver.
	TerraformCloudWorkspace *TerraformCloudWorkspace `json:"terraform_cloud_workspace,omitempty"`

	// TerraformVersion Deprecated, use task.terraform_cloud_workspace.terraform_version instead. Enterprise only. The version of Terraform to use for the Terraform Cloud workspace associated with the task. This is only available when used with the Terraform Cloud driver. Defaults to the latest compatible version if not set.
	TerraformVersion *string `json:"terraform_version,omitempty"`

	// Variables The map of variables that are provided to the task's module.
	Variables *VariableMap `json:"variables,omitempty"`

	// Version The version of the configured module that the task uses. Defaults to the latest version if not set.
	Version *string `json:"version,omitempty"`
}

// TaskDeleteResponse defines model for TaskDeleteResponse.
type TaskDeleteResponse struct {
	Error     *Error    `json:"error,omitempty"`
	RequestId RequestID `json:"request_id"`
}

// TaskRequest defines model for TaskRequest.
type TaskRequest struct {
	Task Task `json:"task"`
}

// TaskResponse defines model for TaskResponse.
type TaskResponse struct {
	Error     *Error    `json:"error,omitempty"`
	RequestId RequestID `json:"request_id"`
	Run       *Run      `json:"run,omitempty"`
	Task      *Task     `json:"task,omitempty"`
}

// TasksResponse defines model for TasksResponse.
type TasksResponse struct {
	RequestId RequestID `json:"request_id"`
	Tasks     *[]Task   `json:"tasks,omitempty"`
}

// TerraformCloudWorkspace Enterprise only. Configuration values to use for the Terraform Cloud workspace associated with the task. This is only available when used with the Terraform Cloud driver.
type TerraformCloudWorkspace struct {
	// AgentPoolId Enterprise only. Agent pool ID to set for the Terraform Cloud workspace associated with the task when the execution mode is "agent". Either agent_pool_id or agent_pool_name is required if execution mode is "agent". If both are set, then agent_pool_id takes precedence.
	AgentPoolId *string `json:"agent_pool_id,omitempty"`

	// AgentPoolName Enterprise only. Name of agent pool to set for the Terraform Cloud workspace associated with the task when the execution mode is "agent". Either agent_pool_id or agent_pool_name is required if execution mode is "agent". If both are set, then agent_pool_id takes precedence.
	AgentPoolName *string `json:"agent_pool_name,omitempty"`

	// ExecutionMode Enterprise only. Execution mode to set for the Terraform Cloud workspace associated with the task. Can only be "remote" or "agent". Defaults to "remote".
	ExecutionMode *string `json:"execution_mode,omitempty"`

	// TerraformVersion Enterprise only. The version of Terraform to use for the Terraform Cloud workspace associated with the task. This is only available when used with the Terraform Cloud driver. Defaults to the latest compatible version if not set.
	TerraformVersion *string `json:"terraform_version,omitempty"`
}

// VariableMap The map of variables that are provided to the task's module.
type VariableMap map[string]string

// CreateTaskParams defines parameters for CreateTask.
type CreateTaskParams struct {
	// Run Different modes for running. Supports run now which runs the task immediately
	// and run inspect which creates a dry run task that is inspected and discarded
	// at the end of the inspection.
	Run *CreateTaskParamsRun `form:"run,omitempty" json:"run,omitempty"`
}

// CreateTaskParamsRun defines parameters for CreateTask.
type CreateTaskParamsRun string

// CreateTaskJSONRequestBody defines body for CreateTask for application/json ContentType.
type CreateTaskJSONRequestBody = TaskRequest
