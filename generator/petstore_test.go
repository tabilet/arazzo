package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tabilet/arazzo/arazzo1"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestPetstoreRoundTrip(t *testing.T) {
	testDir := "testdata"
	arazzoFile := filepath.Join(testDir, "petstore.arazzo.yaml")
	openapiFile := filepath.Join(testDir, "petstore.openapi.yaml")
	genFile := filepath.Join(testDir, "petstore_gen.yaml")

	// 1. Create Generator from Arazzo
	gen, err := NewGeneratorFromArazzo(arazzoFile, openapiFile)
	if err != nil {
		t.Fatalf("NewGeneratorFromArazzo failed: %v", err)
	}

	// 2. Marshal Generator to YAML
	genBytes, err := yaml.Marshal(gen)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}
	if err := os.WriteFile(genFile, genBytes, 0644); err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}
	t.Logf("Generated config written to %s", genFile)

	// 3. Create Arazzo from Generator
	arazzoGen, err := NewArazzoFromFiles(openapiFile, genFile)
	if err != nil {
		t.Fatalf("NewArazzoFromFiles failed: %v", err)
	}

	// 4. Load Original Arazzo for comparison
	origBytes, err := os.ReadFile(arazzoFile)
	if err != nil {
		t.Fatalf("reading original arazzo file: %v", err)
	}
	var arazzoOrig arazzo1.Arazzo
	if err := yaml.Unmarshal(origBytes, &arazzoOrig); err != nil {
		t.Fatalf("parsing original arazzo file: %v", err)
	}

	// 5. Compare
	// We expect differences, so we log them instead of failing immediately,
	// unless the structure is completely wrong.

	// Normalize generated Arazzo by round-tripping through YAML to convert structs to maps (matching original)
	genBytes, _ = yaml.Marshal(arazzoGen)
	var arazzoGenMap arazzo1.Arazzo
	if err := yaml.Unmarshal(genBytes, &arazzoGenMap); err != nil {
		t.Fatalf("normalizing generated arazzo: %v", err)
	}

	// Normalize Source URL in original because we changed the location in the test
	if len(arazzoOrig.SourceDescriptions) > 0 {
		arazzoOrig.SourceDescriptions[0].URL = openapiFile
	}
	if len(arazzoGenMap.SourceDescriptions) > 0 {
		arazzoGenMap.SourceDescriptions[0].URL = openapiFile
	}

	// Ignore fields that are known to be lost or changed
	opts := []cmp.Option{
		// cmpopts.IgnoreFields(arazzo1.Step{}, "Description", "Parameters", "Outputs", "OperationPath"), // Now supported
		// cmpopts.IgnoreFields(arazzo1.Workflow{}, "Inputs", "Outputs", "Description"), // Now supported
		// cmpopts.IgnoreFields(arazzo1.Info{}, "Summary", "Description"), // Info summary/desc still lost (Generator only has Provider Name/URL), but Workflow summary/desc is supported
		// cmpopts.IgnoreFields(arazzo1.Info{}, "Summary", "Description"),
		// cmpopts.IgnoreFields(arazzo1.Info{}, "Summary", "Description", "Title", "Version"),
		// cmpopts.IgnoreFields(arazzo1.SourceDescription{}, "Name", "URL"),  // Now supported (Name) and Normalized (URL)
	}

	if diff := cmp.Diff(&arazzoOrig, &arazzoGenMap, opts...); diff != "" {
		t.Errorf("Petstore RoundTrip comparison mismatch (-want +got):\n%s", diff)

		// Dump generated JSON for manual inspection if needed
		dumpJSON(t, "original_dump.json", &arazzoOrig)
		dumpJSON(t, "generated_dump.json", arazzoGen)
	} else {
		t.Log("Petstore RoundTrip matched (with ignores)!")
	}

	// Sub-test: HCL Input
	t.Run("HCL Input", func(t *testing.T) {
		hclFile := filepath.Join(testDir, "petstore_gen.hcl")
		// Generate from HCL
		arazzoHCL, err := NewArazzoFromFiles(openapiFile, hclFile, "hcl")
		assert.NoError(t, err)
		if err != nil {
			return
		}
		assert.NotNil(t, arazzoHCL)
		if arazzoHCL == nil {
			return
		}

		// Detailed check on new step
		steps := arazzoHCL.Workflows[0].Steps
		assert.Len(t, steps, 3)
		step3 := steps[2]
		assert.Equal(t, "placeOrderStep", step3.StepId)

		// RequestBody check
		assert.NotNil(t, step3.RequestBody)
		t.Logf("DEBUG Payload Type: %T, Value: %+v", step3.RequestBody.Payload, step3.RequestBody.Payload)
		payload, ok := step3.RequestBody.Payload.(map[string]interface{})
		assert.True(t, ok, "Payload should be a map")

		// Check for explicit values from HCL (petId=1, quantity=1)
		// Use EqualValues to handle int vs float64 differences
		assert.EqualValues(t, 1, payload["petId"], "petId should be 1")
		assert.EqualValues(t, 1, payload["quantity"], "quantity should be 1")
	})
}

func dumpJSON(t *testing.T, filename string, v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	_ = os.WriteFile(filepath.Join("testdata", filename), b, 0644)
}
