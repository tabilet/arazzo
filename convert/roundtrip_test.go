package convert

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tabilet/arazzo/arazzo1"
	"gopkg.in/yaml.v3"
)

// TestRoundTripExamples tests JSON-HCL-JSON round-trip conversion for all example files.
func TestRoundTripExamples(t *testing.T) {
	examplesDir := "./examples/1.0.0"

	// Files with known HCL serialization limitations:
	knownLimitations := map[string]string{
		// Empty - all previously known issues have been fixed
	}

	// Find all arazzo YAML files
	files, err := filepath.Glob(filepath.Join(examplesDir, "*.arazzo.yaml"))
	if err != nil {
		t.Fatalf("Failed to glob examples: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No example files found")
	}

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			if reason, hasLimitation := knownLimitations[name]; hasLimitation {
				t.Skipf("Skipping due to known HCL limitation: %s", reason)
			}
			testRoundTripFile(t, file)
		})
	}
}

func testRoundTripFile(t *testing.T, filePath string) {
	// Read YAML file
	yamlData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Parse YAML to Arazzo struct
	var doc1 arazzo1.Arazzo
	if err := yaml.Unmarshal(yamlData, &doc1); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Convert to JSON first (our intermediate format)
	jsonData1, err := json.Marshal(&doc1)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	// JSON -> HCL
	hclData, err := JSONToHCL(jsonData1)
	if err != nil {
		t.Fatalf("Failed to convert JSON to HCL: %v", err)
	}

	t.Logf("HCL output for %s:\n%s", filepath.Base(filePath), string(hclData))

	// HCL -> JSON
	jsonData2, err := HCLToJSON(hclData)
	if err != nil {
		t.Fatalf("Failed to convert HCL to JSON: %v", err)
	}

	// Parse both JSON outputs to compare
	var doc2 arazzo1.Arazzo
	if err := json.Unmarshal(jsonData2, &doc2); err != nil {
		t.Fatalf("Failed to unmarshal round-trip JSON: %v", err)
	}

	// Compare key fields
	compareArazzoDocs(t, &doc1, &doc2)
}

func compareArazzoDocs(t *testing.T, doc1, doc2 *arazzo1.Arazzo) {
	t.Helper()

	if doc1.Arazzo != doc2.Arazzo {
		t.Errorf("Arazzo version mismatch: got %q, want %q", doc2.Arazzo, doc1.Arazzo)
	}

	if doc1.Info != nil && doc2.Info != nil {
		if doc1.Info.Title != doc2.Info.Title {
			t.Errorf("Info.Title mismatch: got %q, want %q", doc2.Info.Title, doc1.Info.Title)
		}
		if doc1.Info.Version != doc2.Info.Version {
			t.Errorf("Info.Version mismatch: got %q, want %q", doc2.Info.Version, doc1.Info.Version)
		}
	} else if (doc1.Info == nil) != (doc2.Info == nil) {
		t.Error("Info presence mismatch")
	}

	if len(doc1.SourceDescriptions) != len(doc2.SourceDescriptions) {
		t.Errorf("SourceDescriptions count mismatch: got %d, want %d",
			len(doc2.SourceDescriptions), len(doc1.SourceDescriptions))
	} else {
		for i := range doc1.SourceDescriptions {
			if doc1.SourceDescriptions[i].Name != doc2.SourceDescriptions[i].Name {
				t.Errorf("SourceDescriptions[%d].Name mismatch: got %q, want %q",
					i, doc2.SourceDescriptions[i].Name, doc1.SourceDescriptions[i].Name)
			}
			if doc1.SourceDescriptions[i].URL != doc2.SourceDescriptions[i].URL {
				t.Errorf("SourceDescriptions[%d].URL mismatch: got %q, want %q",
					i, doc2.SourceDescriptions[i].URL, doc1.SourceDescriptions[i].URL)
			}
		}
	}

	if len(doc1.Workflows) != len(doc2.Workflows) {
		t.Errorf("Workflows count mismatch: got %d, want %d",
			len(doc2.Workflows), len(doc1.Workflows))
	} else {
		for i := range doc1.Workflows {
			w1, w2 := doc1.Workflows[i], doc2.Workflows[i]
			if w1.WorkflowId != w2.WorkflowId {
				t.Errorf("Workflows[%d].WorkflowId mismatch: got %q, want %q",
					i, w2.WorkflowId, w1.WorkflowId)
			}
			if len(w1.Steps) != len(w2.Steps) {
				t.Errorf("Workflows[%d].Steps count mismatch: got %d, want %d",
					i, len(w2.Steps), len(w1.Steps))
			} else {
				for j := range w1.Steps {
					s1, s2 := w1.Steps[j], w2.Steps[j]
					if s1.StepId != s2.StepId {
						t.Errorf("Workflows[%d].Steps[%d].StepId mismatch: got %q, want %q",
							i, j, s2.StepId, s1.StepId)
					}
					if s1.OperationId != s2.OperationId {
						t.Errorf("Workflows[%d].Steps[%d].OperationId mismatch: got %q, want %q",
							i, j, s2.OperationId, s1.OperationId)
					}
				}
			}
		}
	}
}

// TestRoundTripSpecificExample tests a specific example file with detailed output.
func TestRoundTripLoginAndRetrievePets(t *testing.T) {
	filePath := "./examples/1.0.0/LoginAndRetrievePets.arazzo.yaml"

	yamlData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var doc arazzo1.Arazzo
	if err := yaml.Unmarshal(yamlData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Verify parsed correctly
	if doc.Arazzo != "1.0.0" {
		t.Errorf("Expected arazzo 1.0.0, got %s", doc.Arazzo)
	}
	if doc.Info.Title != "A pet purchasing workflow" {
		t.Errorf("Expected title 'A pet purchasing workflow', got %s", doc.Info.Title)
	}
	if len(doc.Workflows) != 1 {
		t.Fatalf("Expected 1 workflow, got %d", len(doc.Workflows))
	}
	if len(doc.Workflows[0].Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(doc.Workflows[0].Steps))
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(&doc, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}
	t.Logf("JSON:\n%s", string(jsonData))

	// Convert to HCL
	hclData, err := JSONToHCL(jsonData)
	if err != nil {
		t.Fatalf("Failed to convert to HCL: %v", err)
	}
	t.Logf("HCL:\n%s", string(hclData))

	// Verify HCL contains expected elements
	hclStr := string(hclData)
	expectedElements := []string{
		`arazzo = "1.0.0"`,
		"info {",
		"sourceDescription",
		"petStoreDescription",
		"workflow",
		"loginUserRetrievePet",
		"step",
		"loginStep",
		"getPetStep",
	}

	for _, elem := range expectedElements {
		if !strings.Contains(hclStr, elem) {
			t.Errorf("HCL output missing expected element: %s", elem)
		}
	}

	// Convert back to JSON
	jsonData2, err := HCLToJSON(hclData)
	if err != nil {
		t.Fatalf("Failed to convert HCL back to JSON: %v", err)
	}

	// Parse and compare
	var doc2 arazzo1.Arazzo
	if err := json.Unmarshal(jsonData2, &doc2); err != nil {
		t.Fatalf("Failed to unmarshal round-trip JSON: %v", err)
	}

	if doc2.Info.Title != doc.Info.Title {
		t.Errorf("Title mismatch after round-trip: got %q, want %q", doc2.Info.Title, doc.Info.Title)
	}
	if doc2.Workflows[0].WorkflowId != doc.Workflows[0].WorkflowId {
		t.Errorf("WorkflowId mismatch after round-trip")
	}
	if len(doc2.Workflows[0].Steps) != len(doc.Workflows[0].Steps) {
		t.Errorf("Steps count mismatch after round-trip")
	}
}

// TestYAMLToHCLDirect tests direct YAML to HCL conversion.
func TestYAMLToHCLDirect(t *testing.T) {
	yamlData := []byte(`
arazzo: "1.0.0"
info:
  title: Test Workflow
  version: "1.0.0"
sourceDescriptions:
  - name: api
    url: ./openapi.json
    type: openapi
workflows:
  - workflowId: test-flow
    steps:
      - stepId: step1
        operationId: getUsers
        successCriteria:
          - condition: $statusCode == 200
`)

	// Parse YAML
	var doc arazzo1.Arazzo
	if err := yaml.Unmarshal(yamlData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Convert to HCL via JSON
	jsonData, err := json.Marshal(&doc)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	hclData, err := JSONToHCL(jsonData)
	if err != nil {
		t.Fatalf("Failed to convert to HCL: %v", err)
	}

	t.Logf("HCL:\n%s", string(hclData))

	// Verify round-trip
	jsonData2, err := HCLToJSON(hclData)
	if err != nil {
		t.Fatalf("Failed to convert back to JSON: %v", err)
	}

	var doc2 arazzo1.Arazzo
	if err := json.Unmarshal(jsonData2, &doc2); err != nil {
		t.Fatalf("Failed to unmarshal round-trip: %v", err)
	}

	if doc2.Arazzo != "1.0.0" {
		t.Errorf("Arazzo version mismatch: %s", doc2.Arazzo)
	}
	if doc2.Info.Title != "Test Workflow" {
		t.Errorf("Title mismatch: %s", doc2.Info.Title)
	}
	if len(doc2.Workflows) != 1 || doc2.Workflows[0].WorkflowId != "test-flow" {
		t.Error("Workflow mismatch")
	}
}

// TestRefTransformation tests that $ref is correctly transformed to _ref for HCL
// and back to $ref when converting back to JSON.
func TestRefTransformation(t *testing.T) {
	// Create a document with $ref in component inputs and workflow inputs
	doc := &arazzo1.Arazzo{
		Arazzo: "1.0.0",
		Info: &arazzo1.Info{
			Title:   "Ref Test",
			Version: "1.0.0",
		},
		SourceDescriptions: []*arazzo1.SourceDescription{
			{
				Name: "api",
				URL:  "./api.json",
			},
		},
		Workflows: []*arazzo1.Workflow{
			{
				WorkflowId: "test-workflow",
				Inputs: map[string]any{
					"$ref": "#/components/inputs/workflow_input",
				},
				Steps: []*arazzo1.Step{
					{
						StepId:      "step1",
						OperationId: "op1",
					},
				},
			},
		},
		Components: &arazzo1.Components{
			Inputs: map[string]any{
				"my_input": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"nested": map[string]any{
							"$ref": "#/components/inputs/nested_ref",
						},
					},
				},
				"simple_ref": map[string]any{
					"$ref": "#/components/inputs/other",
				},
			},
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	// Convert to HCL
	hclData, err := JSONToHCL(jsonData)
	if err != nil {
		t.Fatalf("Failed to convert to HCL: %v", err)
	}

	hclStr := string(hclData)
	t.Logf("HCL output:\n%s", hclStr)

	// Verify _ref appears in HCL (not $ref)
	if strings.Contains(hclStr, "$ref") {
		t.Error("HCL output should not contain $ref, it should be _ref")
	}
	if !strings.Contains(hclStr, "_ref") {
		t.Error("HCL output should contain _ref")
	}

	// Convert back to JSON
	jsonData2, err := HCLToJSON(hclData)
	if err != nil {
		t.Fatalf("Failed to convert back to JSON: %v", err)
	}

	t.Logf("JSON output:\n%s", string(jsonData2))

	// Verify $ref is restored in JSON
	jsonStr := string(jsonData2)
	// Check for "_ref": pattern (the colon distinguishes it from key names like "simple_ref")
	if strings.Contains(jsonStr, `"_ref":`) {
		t.Error("JSON output should not contain \"_ref\": as a key, it should be \"$ref\":")
	}
	if !strings.Contains(jsonStr, `"$ref":`) {
		t.Error("JSON output should contain \"$ref\": as a key")
	}

	// Parse and verify structure
	var doc2 arazzo1.Arazzo
	if err := json.Unmarshal(jsonData2, &doc2); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check component inputs contain $ref
	if doc2.Components == nil || doc2.Components.Inputs == nil {
		t.Fatal("Components.Inputs should not be nil")
	}

	// Check nested $ref was restored
	myInput := doc2.Components.Inputs["my_input"].(map[string]any)
	props := myInput["properties"].(map[string]any)
	nested := props["nested"].(map[string]any)
	if _, ok := nested["$ref"]; !ok {
		t.Error("nested should contain $ref key")
	}

	// Check simple $ref was restored
	simpleRef := doc2.Components.Inputs["simple_ref"].(map[string]any)
	if _, ok := simpleRef["$ref"]; !ok {
		t.Error("simple_ref should contain $ref key")
	}

	// Check workflow inputs $ref was restored
	if doc2.Workflows[0].Inputs == nil {
		t.Error("Workflow inputs should not be nil")
	} else {
		workflowInputs := doc2.Workflows[0].Inputs.(map[string]any)
		if _, ok := workflowInputs["$ref"]; !ok {
			t.Error("Workflow inputs should contain $ref key")
		}
	}
}
