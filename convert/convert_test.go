package convert

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tabilet/arazzo/arazzo1"
)

func TestJSONToHCL(t *testing.T) {
	jsonData := []byte(`{
		"arazzo": "1.0.0",
		"info": {
			"title": "Pet Store Workflow",
			"version": "1.0.0"
		},
		"sourceDescriptions": [
			{
				"name": "petstore",
				"url": "./openapi.json",
				"type": "openapi"
			}
		],
		"workflows": [
			{
				"workflowId": "get-pet",
				"steps": [
					{
						"stepId": "fetch-pet",
						"operationId": "getPetById"
					}
				]
			}
		]
	}`)

	hclData, err := JSONToHCL(jsonData)
	if err != nil {
		t.Fatalf("JSONToHCL failed: %v", err)
	}

	hclStr := string(hclData)

	// Check for expected HCL elements
	if !strings.Contains(hclStr, "arazzo") {
		t.Error("HCL output missing 'arazzo'")
	}
	if !strings.Contains(hclStr, "info") {
		t.Error("HCL output missing 'info' block")
	}
	if !strings.Contains(hclStr, "sourceDescription") {
		t.Error("HCL output missing 'sourceDescription' block")
	}
	if !strings.Contains(hclStr, "workflow") {
		t.Error("HCL output missing 'workflow' block")
	}
}

func TestHCLToJSON(t *testing.T) {
	hclData := []byte(`
arazzo = "1.0.0"

info {
  title   = "Pet Store Workflow"
  version = "1.0.0"
}

sourceDescription "petstore" {
  url  = "./openapi.json"
  type = "openapi"
}

workflow "get-pet" {
  step "fetch-pet" {
    operationId = "getPetById"
  }
}
`)

	jsonData, err := HCLToJSON(hclData)
	if err != nil {
		t.Fatalf("HCLToJSON failed: %v", err)
	}

	// Parse the JSON to verify structure
	var doc arazzo1.Arazzo
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if doc.Arazzo != "1.0.0" {
		t.Errorf("Expected arazzo '1.0.0', got '%s'", doc.Arazzo)
	}
	if doc.Info == nil || doc.Info.Title != "Pet Store Workflow" {
		t.Error("Info not properly converted")
	}
	if len(doc.SourceDescriptions) != 1 || doc.SourceDescriptions[0].Name != "petstore" {
		t.Error("SourceDescriptions not properly converted")
	}
	if len(doc.Workflows) != 1 || doc.Workflows[0].WorkflowId != "get-pet" {
		t.Error("Workflows not properly converted")
	}
}

func TestMarshalUnmarshalHCL(t *testing.T) {
	// Create a document programmatically
	doc := &arazzo1.Arazzo{
		Arazzo: "1.0.0",
		Info: &arazzo1.Info{
			Title:       "Test API",
			Summary:     "A test workflow",
			Description: "This is a test",
			Version:     "1.0.0",
		},
		SourceDescriptions: []*arazzo1.SourceDescription{
			{
				Name: "api",
				URL:  "https://example.com/openapi.json",
				Type: arazzo1.SourceDescriptionTypeOpenAPI,
			},
		},
		Workflows: []*arazzo1.Workflow{
			{
				WorkflowId:  "test-workflow",
				Summary:     "Test workflow",
				Description: "A workflow for testing",
				Steps: []*arazzo1.Step{
					{
						StepId:      "step1",
						OperationId: "getUser",
						SuccessCriteria: []*arazzo1.Criterion{
							{
								Condition: "$statusCode == 200",
								Type:      arazzo1.CriterionTypeSimple,
							},
						},
					},
				},
				Outputs: map[string]string{
					"userId": "$steps.step1.outputs.id",
				},
			},
		},
	}

	// Marshal to HCL
	hclData, err := MarshalHCL(doc)
	if err != nil {
		t.Fatalf("MarshalHCL failed: %v", err)
	}

	// Unmarshal back
	var doc2 arazzo1.Arazzo
	if err := UnmarshalHCL(hclData, &doc2); err != nil {
		t.Fatalf("UnmarshalHCL failed: %v", err)
	}

	// Verify
	if doc2.Arazzo != doc.Arazzo {
		t.Errorf("Arazzo version mismatch: got %s, want %s", doc2.Arazzo, doc.Arazzo)
	}
	if doc2.Info.Title != doc.Info.Title {
		t.Errorf("Title mismatch: got %s, want %s", doc2.Info.Title, doc.Info.Title)
	}
	if len(doc2.Workflows) != len(doc.Workflows) {
		t.Errorf("Workflow count mismatch: got %d, want %d", len(doc2.Workflows), len(doc.Workflows))
	}
}

func TestComplexWorkflowConversion(t *testing.T) {
	jsonData := []byte(`{
		"arazzo": "1.0.0",
		"info": {
			"title": "Complex Workflow",
			"version": "2.0.0",
			"summary": "A complex test workflow",
			"description": "Testing all features"
		},
		"sourceDescriptions": [
			{"name": "api1", "url": "./api1.json", "type": "openapi"},
			{"name": "api2", "url": "./api2.json", "type": "arazzo"}
		],
		"workflows": [
			{
				"workflowId": "main-flow",
				"summary": "Main workflow",
				"dependsOn": ["setup-flow"],
				"steps": [
					{
						"stepId": "create",
						"operationId": "createResource",
						"successCriteria": [
							{"condition": "$statusCode == 201", "type": "simple"}
						],
						"outputs": {
							"resourceId": "$response.body.id"
						}
					},
					{
						"stepId": "verify",
						"operationId": "getResource"
					}
				],
				"outputs": {
					"result": "$steps.verify.outputs.body"
				}
			}
		]
	}`)

	// Convert to HCL
	hclData, err := JSONToHCL(jsonData)
	if err != nil {
		t.Fatalf("JSONToHCL failed: %v", err)
	}

	t.Logf("Generated HCL:\n%s", string(hclData))

	// Convert back to JSON
	jsonData2, err := HCLToJSON(hclData)
	if err != nil {
		t.Fatalf("HCLToJSON failed: %v", err)
	}

	// Parse both and compare
	var doc1, doc2 arazzo1.Arazzo
	if err := json.Unmarshal(jsonData, &doc1); err != nil {
		t.Fatalf("Failed to parse original JSON: %v", err)
	}
	if err := json.Unmarshal(jsonData2, &doc2); err != nil {
		t.Fatalf("Failed to parse converted JSON: %v", err)
	}

	// Compare key fields
	if doc1.Arazzo != doc2.Arazzo {
		t.Errorf("Arazzo version mismatch after round-trip")
	}
	if doc1.Info.Title != doc2.Info.Title {
		t.Errorf("Title mismatch after round-trip")
	}
	if len(doc1.SourceDescriptions) != len(doc2.SourceDescriptions) {
		t.Errorf("SourceDescriptions count mismatch after round-trip")
	}
	if len(doc1.Workflows) != len(doc2.Workflows) {
		t.Errorf("Workflows count mismatch after round-trip")
	}
	if len(doc1.Workflows[0].Steps) != len(doc2.Workflows[0].Steps) {
		t.Errorf("Steps count mismatch after round-trip")
	}
}

func TestRoundTripPreservesCriteriaAndReplacements(t *testing.T) {
	doc := &arazzo1.Arazzo{
		Arazzo: "1.0.0",
		Info: &arazzo1.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		SourceDescriptions: []*arazzo1.SourceDescription{
			{
				Name: "api",
				URL:  "./openapi.json",
				Type: arazzo1.SourceDescriptionTypeOpenAPI,
			},
		},
		Workflows: []*arazzo1.Workflow{
			{
				WorkflowId: "workflow",
				Steps: []*arazzo1.Step{
					{
						StepId:      "step1",
						OperationId: "getUser",
						SuccessCriteria: []*arazzo1.Criterion{
							{
								Condition: "$.status",
								ExpressionType: &arazzo1.CriterionExpressionType{
									Type:    arazzo1.CriterionTypeJSONPath,
									Version: "draft-goessner-dispatch-jsonpath-00",
								},
							},
						},
						OnSuccess: []*arazzo1.SuccessActionOrReusable{
							{
								SuccessAction: &arazzo1.SuccessAction{
									Name:   "continue",
									Type:   arazzo1.SuccessActionTypeGoto,
									StepId: "nextStep",
									Criteria: []*arazzo1.Criterion{
										{
											Condition: "$statusCode == 200",
											Type:      arazzo1.CriterionTypeSimple,
										},
									},
								},
							},
						},
						RequestBody: &arazzo1.RequestBody{
							ContentType: "application/json",
							Payload: map[string]any{
								"name": "test",
							},
							Replacements: []*arazzo1.PayloadReplacement{
								{
									Target: "/name",
									Value:  "override",
								},
							},
						},
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	hclData, err := JSONToHCL(jsonData)
	if err != nil {
		t.Fatalf("JSONToHCL failed: %v", err)
	}

	jsonData2, err := HCLToJSON(hclData)
	if err != nil {
		t.Fatalf("HCLToJSON failed: %v", err)
	}

	var doc2 arazzo1.Arazzo
	if err := json.Unmarshal(jsonData2, &doc2); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	step := doc2.Workflows[0].Steps[0]
	if len(step.SuccessCriteria) != 1 {
		t.Fatalf("Expected 1 success criterion, got %d", len(step.SuccessCriteria))
	}
	if step.SuccessCriteria[0].ExpressionType == nil {
		t.Fatal("Expected expression type to be preserved")
	}
	if step.SuccessCriteria[0].ExpressionType.Version != "draft-goessner-dispatch-jsonpath-00" {
		t.Errorf("Expected expression type version to be preserved, got %q", step.SuccessCriteria[0].ExpressionType.Version)
	}

	if len(step.OnSuccess) != 1 || step.OnSuccess[0].SuccessAction == nil {
		t.Fatal("Expected onSuccess action to be preserved")
	}
	if len(step.OnSuccess[0].SuccessAction.Criteria) != 1 {
		t.Fatalf("Expected 1 onSuccess criterion, got %d", len(step.OnSuccess[0].SuccessAction.Criteria))
	}

	if step.RequestBody == nil {
		t.Fatal("Expected requestBody to be preserved")
	}
	if len(step.RequestBody.Replacements) != 1 {
		t.Fatalf("Expected 1 replacement, got %d", len(step.RequestBody.Replacements))
	}
	if step.RequestBody.Replacements[0].Target != "/name" {
		t.Errorf("Expected replacement target '/name', got %q", step.RequestBody.Replacements[0].Target)
	}
}

func TestHCLConversionPreservesDollarKeys(t *testing.T) {
	jsonData := []byte(`{
		"arazzo": "1.0.0",
		"info": {
			"title": "Dollar Key Workflow",
			"version": "1.0.0"
		},
		"sourceDescriptions": [
			{
				"name": "api",
				"url": "./openapi.json",
				"type": "openapi"
			}
		],
		"workflows": [
			{
				"workflowId": "workflow",
				"inputs": {
					"$custom": {
						"type": "string"
					},
					"regular": {
						"type": "string"
					}
				},
				"steps": [
					{
						"stepId": "step1",
						"operationId": "getUser",
						"requestBody": {
							"contentType": "application/json",
							"payload": {
								"$meta": {
									"id": 1
								},
								"name": "test"
							}
						}
					}
				]
			}
		]
	}`)

	hclData, err := JSONToHCL(jsonData)
	if err != nil {
		t.Fatalf("JSONToHCL failed: %v", err)
	}

	hclStr := string(hclData)
	if !strings.Contains(hclStr, "__dollar__custom") {
		t.Error("HCL output missing transformed $custom key")
	}
	if !strings.Contains(hclStr, "__dollar__meta") {
		t.Error("HCL output missing transformed $meta key")
	}

	jsonData2, err := HCLToJSON(hclData)
	if err != nil {
		t.Fatalf("HCLToJSON failed: %v", err)
	}

	var doc arazzo1.Arazzo
	if err := json.Unmarshal(jsonData2, &doc); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	inputs, ok := doc.Workflows[0].Inputs.(map[string]any)
	if !ok {
		t.Fatalf("Expected inputs to be map[string]any, got %T", doc.Workflows[0].Inputs)
	}
	if _, ok := inputs["$custom"]; !ok {
		t.Error("Expected $custom key in inputs after round-trip")
	}

	step := doc.Workflows[0].Steps[0]
	if step.RequestBody == nil {
		t.Fatal("Expected requestBody to be preserved")
	}
	payload, ok := step.RequestBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("Expected payload to be map[string]any, got %T", step.RequestBody.Payload)
	}
	if _, ok := payload["$meta"]; !ok {
		t.Error("Expected $meta key in payload after round-trip")
	}
}

func TestRoundTripPreservesReusableActions(t *testing.T) {
	doc := &arazzo1.Arazzo{
		Arazzo: "1.0.0",
		Info: &arazzo1.Info{
			Title:   "Reusable Actions",
			Version: "1.0.0",
		},
		SourceDescriptions: []*arazzo1.SourceDescription{
			{
				Name: "api",
				URL:  "./openapi.json",
				Type: arazzo1.SourceDescriptionTypeOpenAPI,
			},
		},
		Workflows: []*arazzo1.Workflow{
			{
				WorkflowId: "workflow",
				Steps: []*arazzo1.Step{
					{
						StepId:      "step1",
						OperationId: "getUser",
						OnSuccess: []*arazzo1.SuccessActionOrReusable{
							{
								Reusable: &arazzo1.ReusableObject{
									Reference: "$components.successActions.LogSuccess",
								},
							},
						},
						OnFailure: []*arazzo1.FailureActionOrReusable{
							{
								Reusable: &arazzo1.ReusableObject{
									Reference: "$components.failureActions.RetryOnce",
									Value: map[string]any{
										"retryLimit": 5,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	hclData, err := JSONToHCL(jsonData)
	if err != nil {
		t.Fatalf("JSONToHCL failed: %v", err)
	}

	jsonData2, err := HCLToJSON(hclData)
	if err != nil {
		t.Fatalf("HCLToJSON failed: %v", err)
	}

	var doc2 arazzo1.Arazzo
	if err := json.Unmarshal(jsonData2, &doc2); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	step := doc2.Workflows[0].Steps[0]
	if len(step.OnSuccess) != 1 || step.OnSuccess[0].Reusable == nil {
		t.Fatal("Expected reusable onSuccess action to be preserved")
	}
	if step.OnSuccess[0].Reusable.Reference != "$components.successActions.LogSuccess" {
		t.Errorf("Expected onSuccess reusable reference to be preserved, got %q", step.OnSuccess[0].Reusable.Reference)
	}
	if len(step.OnFailure) != 1 || step.OnFailure[0].Reusable == nil {
		t.Fatal("Expected reusable onFailure action to be preserved")
	}
	if step.OnFailure[0].Reusable.Reference != "$components.failureActions.RetryOnce" {
		t.Errorf("Expected onFailure reusable reference to be preserved, got %q", step.OnFailure[0].Reusable.Reference)
	}
	if step.OnFailure[0].Reusable.Value == nil {
		t.Fatal("Expected onFailure reusable value override to be preserved")
	}
}

func TestHCLToJSONIndent(t *testing.T) {
	hclData := []byte(`
arazzo = "1.0.0"

info {
  title   = "Test"
  version = "1.0.0"
}

sourceDescription "api" {
  url = "./api.json"
}

workflow "test" {
  step "s1" {
    operationId = "op1"
  }
}
`)

	jsonData, err := HCLToJSONIndent(hclData, "", "  ")
	if err != nil {
		t.Fatalf("HCLToJSONIndent failed: %v", err)
	}

	// Verify it's indented
	if !strings.Contains(string(jsonData), "\n") {
		t.Error("JSON output is not indented")
	}

	// Verify it's valid JSON
	var doc arazzo1.Arazzo
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}
}
