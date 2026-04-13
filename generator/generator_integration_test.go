package generator

import (
	"testing"

	"github.com/tabilet/arazzo/arazzo1"
	"github.com/tabilet/oas/openapi31"
	"github.com/stretchr/testify/assert"
)

// TestGenerator_ToArazzo_ParametersIntegration verifies that parameters defined in the Generator config
// (specifically optional string requests) are correctly processed and enriched in the final Arazzo output.
func TestGenerator_ToArazzo_ParametersIntegration(t *testing.T) {
	// 1. Setup Mock OpenAPI Doc
	doc := &openapi31.OpenAPI{
		Info: &openapi31.Info{
			Title: "Test API",
		},
		Servers: []*openapi31.Server{
			{URL: "http://api.example.com"},
		},
		Paths: &openapi31.Paths{
			Paths: map[string]*openapi31.PathItem{
				"/items": {
					Get: &openapi31.Operation{
						OperationID: "listItems",
						Parameters: []*openapi31.Parameter{
							{
								Name:     "limit",
								In:       "query",
								Required: true, // Should be auto-included
							},
							{
								Name:     "offset",
								In:       "query",
								Required: false, // Optional, needs explicit request
							},
						},
					},
				},
			},
		},
	}

	// 2. Setup Generator Config
	gen := &Generator{
		openapiDoc: doc,
		Provider: &Provider{
			Name: "test-provider",
		},
		Workflows: []*WorkflowSpec{
			{
				WorkflowId: "wf-1",
				Summary:    "Test Workflow",
				Steps: []*OperationSpec{
					{
						Name:        "step-1",
						OperationId: "listItems",
						Parameters: []interface{}{
							"offset", // Explicitly requesting the optional 'offset' parameter
						},
					},
				},
			},
		},
	}

	// 3. Execute ToArazzo
	az, err := gen.ToArazzo("test.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, az)
	assert.Len(t, az.Workflows, 1)
	assert.Len(t, az.Workflows[0].Steps, 1)

	step := az.Workflows[0].Steps[0]
	assert.Equal(t, "step-1", step.StepId)

	// 4. Verify Parameters
	// Expecting 2 parameters: 'limit' (auto-required) and 'offset' (requested string)
	assert.Len(t, step.Parameters, 2)

	paramMap := make(map[string]*arazzo1.Parameter)
	for _, p := range step.Parameters {
		if pStruct, ok := p.(*arazzo1.Parameter); ok {
			paramMap[pStruct.Name] = pStruct
		}
	}

	// Check 'limit'
	limit, ok := paramMap["limit"]
	assert.True(t, ok, "limit parameter should be auto-included")
	assert.Equal(t, arazzo1.ParameterInQuery, limit.In)
	assert.Equal(t, "$inputs.limit", limit.Value)

	// Check 'offset'
	offset, ok := paramMap["offset"]
	assert.True(t, ok, "offset parameter should be included because it was requested")
	assert.Equal(t, arazzo1.ParameterInQuery, offset.In)
	assert.Equal(t, "$inputs.offset", offset.Value)
}
