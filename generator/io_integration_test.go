package generator

import (
	"encoding/json"
	"testing"

	"github.com/tabilet/arazzo/arazzo1"
	"github.com/tabilet/oas/openapi31"
	"github.com/stretchr/testify/assert"
)

// TestGenerator_ToArazzo_InputsOutputsIntegration verifies that Inputs and Outputs defined in the Generator config
// are correctly passed through to the final Arazzo workflow.
func TestGenerator_ToArazzo_InputsOutputsIntegration(t *testing.T) {
	// 1. Setup Mock OpenAPI Doc (minimal)
	doc := &openapi31.OpenAPI{
		Info: &openapi31.Info{Title: "Test API"},
		Paths: &openapi31.Paths{
			Paths: map[string]*openapi31.PathItem{
				"/echo": {
					Get: &openapi31.Operation{
						OperationID: "echo",
					},
				},
			},
		},
	}

	// 2. Setup Generator Config with Inputs and Outputs
	inputsJSON := `{"type": "object", "properties": {"userId": {"type": "integer"}}}`
	var inputsObj interface{}
	json.Unmarshal([]byte(inputsJSON), &inputsObj)

	outputsMap := map[string]string{
		"result": "$steps.step-1.outputs.data",
	}

	gen := &Generator{
		openapiDoc: doc,
		Provider:   &Provider{Name: "test"},
		Workflows: []*WorkflowSpec{
			{
				WorkflowId: "wf-io",
				Inputs:     inputsObj,
				Outputs:    outputsMap,
				Steps: []*OperationSpec{
					{
						Name:        "step-1",
						OperationId: "echo",
					},
				},
			},
		},
	}

	// 3. Execute ToArazzo
	az, err := gen.ToArazzo("test.yaml")
	assert.NoError(t, err)

	// 4. Verify Inputs/Outputs in Arazzo
	wf := az.Workflows[0]

	// Verify Inputs
	assert.NotNil(t, wf.Inputs)
	inputsMap, ok := wf.Inputs.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "object", inputsMap["type"])
	props := inputsMap["properties"].(map[string]interface{})
	assert.NotNil(t, props["userId"])

	// Verify Outputs
	assert.Equal(t, outputsMap, wf.Outputs)
	assert.Equal(t, "$steps.step-1.outputs.data", wf.Outputs["result"])
}

// TestGenerator_ToArazzo_DataAssignments verifies that users can explicitly assign values to
// various parameter types (path, header, cookie) and the request body, overriding defaults.
func TestGenerator_ToArazzo_DataAssignments(t *testing.T) {
	// 1. Setup Mock OpenAPI Doc
	doc := &openapi31.OpenAPI{
		Info: &openapi31.Info{Title: "Test API"},
		Paths: &openapi31.Paths{
			Paths: map[string]*openapi31.PathItem{
				"/resource/{id}": {
					Post: &openapi31.Operation{
						OperationID: "updateResource",
						Parameters: []*openapi31.Parameter{
							{Name: "id", In: "path", Required: true},
							{Name: "X-Auth-Token", In: "header", Required: true},
							{Name: "session_id", In: "cookie", Required: false},
						},
						RequestBody: &openapi31.RequestBody{
							Content: map[string]*openapi31.MediaType{
								"application/json": {
									Schema: &openapi31.Schema{},
								},
							},
						},
					},
				},
			},
		},
	}

	// 2. Setup Generator Config with Explicit Values
	gen := &Generator{
		openapiDoc: doc,
		Provider:   &Provider{Name: "test"},
		Workflows: []*WorkflowSpec{
			{
				WorkflowId: "wf-data",
				Steps: []*OperationSpec{
					{
						Name:        "step-override",
						OperationId: "updateResource",
						Parameters: []interface{}{
							// Path Param: Assigning explicit integer
							map[string]interface{}{
								"name":  "id",
								"in":    "path",
								"value": 999,
							},
							// Header Param: Assigning from input variable
							map[string]interface{}{
								"name":  "X-Auth-Token",
								"in":    "header",
								"value": "$inputs.token",
							},
							// Cookie Param: Optional, assigning explicit string
							map[string]interface{}{
								"name":  "session_id",
								"in":    "cookie",
								"value": "sess_abc123",
							},
						},
						// RequestBody: Explicit payload assignment
						RequestBody: map[string]interface{}{
							"contentType": "application/json",
							"payload":     `{"status": "active"}`,
						},
					},
				},
			},
		},
	}

	// 3. Execute ToArazzo
	az, err := gen.ToArazzo("test.yaml")
	assert.NoError(t, err)

	step := az.Workflows[0].Steps[0]

	// 4. Verify Parameter Assignments
	paramMap := make(map[string]*arazzo1.Parameter)
	for _, p := range step.Parameters {
		if pStruct, ok := p.(*arazzo1.Parameter); ok {
			paramMap[pStruct.Name] = pStruct
		} else if pMap, ok := p.(map[string]interface{}); ok {
			name := pMap["name"].(string)
			// Convert to struct for easier assertion
			paramMap[name] = &arazzo1.Parameter{
				Name:  name,
				In:    arazzo1.ParameterIn(pMap["in"].(string)),
				Value: pMap["value"],
			}
		}
	}

	// Assertions
	assert.Equal(t, 999, paramMap["id"].Value)
	assert.Equal(t, "$inputs.token", paramMap["X-Auth-Token"].Value)
	assert.Equal(t, "sess_abc123", paramMap["session_id"].Value)

	// 5. Verify Request Body
	assert.NotNil(t, step.RequestBody)
	assert.Equal(t, "application/json", step.RequestBody.ContentType)
	assert.Equal(t, `{"status": "active"}`, step.RequestBody.Payload)
}
