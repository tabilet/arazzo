package generator

import (
	"github.com/tabilet/arazzo/arazzo1"
	"github.com/tabilet/oas/openapi31"
)

// Generator represents a generator config.
type Generator struct {
	Provider   *Provider           `yaml:"provider" json:"provider" hcl:"provider,block"`
	Workflows  []*WorkflowSpec     `yaml:"workflows" json:"workflows" hcl:"workflow,block"`
	Components *arazzo1.Components `yaml:"components,omitempty" json:"components,omitempty" hcl:"components,block"`
	Extensions map[string]any      `yaml:"extensions,omitempty" json:"extensions,omitempty" hcl:"extensions,optional"`

	// Internal
	openapiDoc *openapi31.OpenAPI
}

// Provider represents the provider configuration.
type Provider struct {
	Name       string                 `yaml:"name" json:"name" hcl:"name"`
	ServerURL  string                 `yaml:"server_url" json:"server_url" hcl:"server_url"`
	Appendices map[string]interface{} `yaml:"appendices" json:"appendices" hcl:"appendices,optional"` // Reserves Info details
	Extensions map[string]any         `yaml:"extensions,omitempty" json:"extensions,omitempty" hcl:"extensions,optional"`
}

// WorkflowSpec defines a workflow in the generator.
type WorkflowSpec struct {
	WorkflowId  string `yaml:"workflow_id" json:"workflowId" hcl:"workflow_id,label"`
	Summary     string `yaml:"summary" json:"summary" hcl:"summary,optional"`
	Description string `yaml:"description" json:"description" hcl:"description,optional"`
	// Inputs defines the runtime arguments for the workflow as a JSON Schema.
	// The generator assumes that required OpenAPI parameters will be available as inputs (e.g., $inputs.myParam).
	Inputs interface{} `yaml:"inputs" json:"inputs" hcl:"inputs,block"`
	// Outputs defines the values returned by the workflow, typically mapped from step outputs using runtime expressions (e.g. "$steps.myStep.outputs.id").
	Outputs map[string]string `yaml:"outputs" json:"outputs" hcl:"outputs,block"`
	// DependsOn specifies a list of other workflows (by their workflowId) that must successfully complete before this workflow can start.
	DependsOn []string `yaml:"depends_on" json:"dependsOn" hcl:"depends_on,optional"`
	// Parameters are not included in WorkflowSpec because the generator focuses on server-defined parameters (OpenAPI).
	// Workflow-level parameters are not strictly defined by the server and should be handled via inputs or manually added if needed.
	// SuccessActions defines actions to take when the workflow completes successfully (e.g. emit an event, loop).
	SuccessActions []*arazzo1.SuccessActionOrReusable `yaml:"success_actions" json:"successActions" hcl:"success_action,block"`
	// FailureActions defines actions to take when the workflow fails (e.g. emit an event, retry).
	FailureActions []*arazzo1.FailureActionOrReusable `yaml:"failure_actions" json:"failureActions" hcl:"failure_action,block"`
	Steps          []*OperationSpec                   `yaml:"steps" json:"steps" hcl:"step,block"`
	Extensions     map[string]any                     `yaml:"extensions,omitempty" json:"extensions,omitempty" hcl:"extensions,optional"`
}

// OperationSpec defines an operation to be included in the Arazzo workflow.
type OperationSpec struct {
	Name string `yaml:"name" json:"name" hcl:"name,label"` // Acts as label in HCL/YAML list item

	// High Fidelity Fields
	Description string `yaml:"description" json:"description" hcl:"description,optional"`
	// Parameters can be:
	// 1. A string: The name of an optional parameter to be auto-filled from OpenAPI (e.g., "trace_id").
	//    Note: Mandatory parameters (Required: true) from OpenAPI are ALWAYS auto-included implicitly.
	// 2. A map/object: A fully defined Arazzo parameter (e.g., {name: "id", in: "path", value: "123"}).
	Parameters []interface{} `yaml:"parameters" json:"parameters" hcl:"parameter,block"`
	// RequestBody defines the payload for the operation.
	// It can be:
	// 1. Raw data (string/object): Used as the explicit payload (contentType inferred from OpenAPI).
	//    Example: { "username": "foo", "pwd": "bar" }
	// 2. A map/object with a "payload" key: Treated as a full RequestBody configuration (allows setting contentType, replacements).
	//    Example: { "contentType": "application/json", "payload": "{...}" }
	// 3. Nil/Empty: Auto-generated from OpenAPI examples.
	// Note: For HCL compatibility, this must be a map. Raw strings are not supported in HCL generator input.
	RequestBody map[string]interface{} `yaml:"request_body" json:"requestBody" hcl:"request_body,optional"`
	// SuccessCriteria defines conditions for step success.
	// 1. If provided: Used as is.
	// 2. If Empty: Auto-generated from OpenAPI 2xx response codes (e.g., "$statusCode == 200").
	SuccessCriteria []*arazzo1.Criterion `yaml:"success_criteria" json:"successCriteria" hcl:"success_criterion,block"`
	// OnSuccess defines actions for successful steps. Default: Continue to next step.
	OnSuccess []*arazzo1.SuccessActionOrReusable `yaml:"on_success" json:"onSuccess" hcl:"on_success,block"`
	// OnFailure defines actions for failed steps. Default: Stop workflow and error.
	OnFailure []*arazzo1.FailureActionOrReusable `yaml:"on_failure" json:"onFailure" hcl:"on_failure,block"`
	// Outputs defines values to extract from this step's result (e.g., "$response.body.id") to be used by subsequent steps.
	Outputs       map[string]string `yaml:"outputs" json:"outputs" hcl:"outputs,block"`
	OperationPath string            `yaml:"operation_path" json:"operationPath" hcl:"operation_path,optional"`
	OperationId   string            `yaml:"operation_id" json:"operationId" hcl:"operation_id,optional"`
	WorkflowId    string            `yaml:"workflow_id" json:"workflowId" hcl:"workflow_id,optional"`
	Extensions    map[string]any    `yaml:"extensions,omitempty" json:"extensions,omitempty" hcl:"extensions,optional"`
}
