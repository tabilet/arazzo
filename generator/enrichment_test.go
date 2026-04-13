package generator

import (
	"testing"

	"github.com/tabilet/arazzo/arazzo1"
	"github.com/tabilet/oas/openapi31"
	"github.com/stretchr/testify/assert"
)

func TestEnrichment_AutoIncludeRequired(t *testing.T) {
	// Setup Mock OpenAPI
	doc := &openapi31.OpenAPI{
		Paths: &openapi31.Paths{
			Paths: map[string]*openapi31.PathItem{
				"/users/{id}": {
					Get: &openapi31.Operation{
						OperationID: "getUser",
						Parameters: []*openapi31.Parameter{
							{
								Name:     "id",
								In:       "path",
								Required: true,
							},
							{
								Name:     "debug",
								In:       "query",
								Required: false,
							},
						},
					},
				},
			},
		},
	}

	// Setup Step pointing to this operation
	step := &arazzo1.Step{
		OperationId: "getUser",
		Parameters:  nil, // No constraints provided
	}

	// Execute
	enrichStepFromOpenAPI(step, doc)

	// Verify
	assert.Len(t, step.Parameters, 1, "Should have 1 auto-included parameter")
	p, ok := step.Parameters[0].(*arazzo1.Parameter)
	assert.True(t, ok, "Should be a Parameter struct")
	assert.Equal(t, "id", p.Name)
	assert.Equal(t, arazzo1.ParameterInPath, p.In)
	assert.Equal(t, "$inputs.id", p.Value)
}

func TestEnrichment_IncludeOptionalByString(t *testing.T) {
	// Setup Mock OpenAPI
	doc := &openapi31.OpenAPI{
		Paths: &openapi31.Paths{
			Paths: map[string]*openapi31.PathItem{
				"/users": {
					Get: &openapi31.Operation{
						OperationID: "listUsers",
						Parameters: []*openapi31.Parameter{
							{
								Name:     "trace_id",
								In:       "header",
								Required: false,
							},
						},
					},
				},
			},
		},
	}

	// Setup Step requesting the optional parameter by name
	step := &arazzo1.Step{
		OperationId: "listUsers",
		Parameters:  []interface{}{"trace_id"},
	}

	// Execute
	enrichStepFromOpenAPI(step, doc)

	// Verify
	assert.Len(t, step.Parameters, 1)
	p, ok := step.Parameters[0].(*arazzo1.Parameter)
	assert.True(t, ok)
	assert.Equal(t, "trace_id", p.Name)
	assert.Equal(t, arazzo1.ParameterInHeader, p.In)
}

func TestEnrichment_SkipDeprecatedDefault(t *testing.T) {
	doc := &openapi31.OpenAPI{
		Paths: &openapi31.Paths{
			Paths: map[string]*openapi31.PathItem{
				"/old": {
					Get: &openapi31.Operation{
						OperationID: "oldOp",
						Parameters: []*openapi31.Parameter{
							{
								Name:       "legacy",
								In:         "query",
								Required:   true,
								Deprecated: true,
							},
						},
					},
				},
			},
		},
	}

	step := &arazzo1.Step{
		OperationId: "oldOp",
	}

	enrichStepFromOpenAPI(step, doc)
	assert.Empty(t, step.Parameters, "Deprecated required params should be skipped by default")
}
