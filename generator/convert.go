package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tabilet/arazzo/arazzo1"
	"github.com/genelet/horizon/dethcl"
	"github.com/tabilet/oas/openapi31"
	"gopkg.in/yaml.v3"
)

// parseOpenAPI parses an OpenAPI file, handling both JSON and YAML.
// Since openapi31 relies on UnmarshalJSON for custom logic, we convert YAML to JSON first.
func parseOpenAPI(content []byte) (*openapi31.OpenAPI, error) {
	// 1. Try JSON directly
	var doc openapi31.OpenAPI
	if err := json.Unmarshal(content, &doc); err == nil {
		return &doc, nil
	}

	// 2. Try YAML -> Interface -> JSON -> Struct
	var obj interface{}
	if err := yaml.Unmarshal(content, &obj); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}

	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("converting yaml to json: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &doc); err != nil {
		return nil, fmt.Errorf("parsing converted json: %w", err)
	}

	return &doc, nil
}

// NewArazzoFromFiles creates an Arazzo document from OpenAPI and Generator files.
func NewArazzoFromFiles(openapiFile, generatorFile string, format ...string) (*arazzo1.Arazzo, error) {
	// Parse Generator
	genBytes, err := os.ReadFile(generatorFile)
	if err != nil {
		return nil, fmt.Errorf("reading generator file: %w", err)
	}
	var gen Generator

	fmtType := "yaml"
	if len(format) > 0 {
		fmtType = format[0]
	}

	switch fmtType {
	case "json":
		if err := json.Unmarshal(genBytes, &gen); err != nil {
			return nil, fmt.Errorf("parsing generator file (json): %w", err)
		}
	case "hcl":
		if err := dethcl.Unmarshal(genBytes, &gen); err != nil {
			return nil, fmt.Errorf("parsing generator file (hcl): %w", err)
		}
	default: // yaml
		if err := yaml.Unmarshal(genBytes, &gen); err != nil {
			return nil, fmt.Errorf("parsing generator file (yaml): %w", err)
		}
	}

	// Parse OpenAPI
	oaBytes, err := os.ReadFile(openapiFile)
	if err != nil {
		return nil, fmt.Errorf("reading openapi file: %w", err)
	}
	doc, err := parseOpenAPI(oaBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing openapi file: %w", err)
	}
	gen.openapiDoc = doc

	return gen.ToArazzo(openapiFile)
}

// NewGeneratorFromArazzo creates a Generator config from Arazzo and OpenAPI files.
func NewGeneratorFromArazzo(arazzoFile, openapiFile string) (*Generator, error) {
	// Parse Arazzo
	azBytes, err := os.ReadFile(arazzoFile)
	if err != nil {
		return nil, fmt.Errorf("reading arazzo file: %w", err)
	}
	var az arazzo1.Arazzo
	// Arazzo can be JSON or YAML
	if err := json.Unmarshal(azBytes, &az); err != nil {
		if err := yaml.Unmarshal(azBytes, &az); err != nil {
			return nil, fmt.Errorf("parsing arazzo file: %w", err)
		}
	}

	// Parse OpenAPI
	oaBytes, err := os.ReadFile(openapiFile)
	if err != nil {
		return nil, fmt.Errorf("reading openapi file: %w", err)
	}
	doc, err := parseOpenAPI(oaBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing openapi file: %w", err)
	}

	gen := &Generator{
		openapiDoc: doc,
		Provider: &Provider{
			Name:      "my-source", // Default
			ServerURL: "",          // Default empty
		},
		Components: az.Components,
		Extensions: az.Extensions,
	}

	if len(doc.Servers) > 0 {
		gen.Provider.ServerURL = doc.Servers[0].URL
	}

	// Try to get source name from Arazzo if possible
	if len(az.SourceDescriptions) > 0 {
		gen.Provider.Name = az.SourceDescriptions[0].Name
	}

	// Reserve Info fields in Appendices
	if az.Info != nil {
		gen.Provider.Appendices = make(map[string]interface{})
		gen.Provider.Appendices["info_title"] = az.Info.Title
		gen.Provider.Appendices["info_version"] = az.Info.Version
		gen.Provider.Appendices["info_summary"] = az.Info.Summary
		gen.Provider.Appendices["info_description"] = az.Info.Description
	}

	// Iterate workflows
	for _, wf := range az.Workflows {
		spec := &WorkflowSpec{
			WorkflowId:     wf.WorkflowId,
			Summary:        wf.Summary,
			Description:    wf.Description,
			Inputs:         wf.Inputs,
			Outputs:        wf.Outputs,
			DependsOn:      wf.DependsOn,
			SuccessActions: wf.SuccessActions,
			FailureActions: wf.FailureActions,
			Extensions:     wf.Extensions,
		}

		for _, step := range wf.Steps {
			opID := step.OperationId
			if idx := strings.LastIndex(opID, "."); idx != -1 {
				opID = opID[idx+1:]
			}

			// Infer name
			name := step.StepId
			if name == "" {
				name = step.OperationId
			}

			op := &OperationSpec{
				Name:            name,
				Description:     step.Description,
				OperationPath:   step.OperationPath,
				OperationId:     step.OperationId,
				WorkflowId:      step.WorkflowId,
				Extensions:      step.Extensions,
				SuccessCriteria: step.SuccessCriteria,
				OnSuccess:       step.OnSuccess,
				OnFailure:       step.OnFailure,
				Outputs:         step.Outputs,
			}

			// Copy parameters
			if len(step.Parameters) > 0 {
				op.Parameters = make([]interface{}, len(step.Parameters))
				copy(op.Parameters, step.Parameters)
			}

			// Copy RequestBody
			if step.RequestBody != nil {
				b, _ := json.Marshal(step.RequestBody)
				var rbMap map[string]interface{}
				_ = json.Unmarshal(b, &rbMap)
				op.RequestBody = rbMap
			}

			spec.Steps = append(spec.Steps, op)
		}
		gen.Workflows = append(gen.Workflows, spec)
	}

	return gen, nil
}

// ToArazzo converts the generator configuration and OpenAPI document to an Arazzo object.
func (g *Generator) ToArazzo(openapiFilename string) (*arazzo1.Arazzo, error) {
	if g.openapiDoc == nil {
		return nil, fmt.Errorf("openapi document not set")
	}

	// Create Arazzo root
	arazzo := &arazzo1.Arazzo{
		Arazzo: "1.0.0",
		Info: &arazzo1.Info{
			Title:   "Generated Arazzo from " + g.openapiDoc.Info.Title,
			Version: "1.0.0",
			Summary: "Generated from " + openapiFilename,
		},
		SourceDescriptions: []*arazzo1.SourceDescription{
			{
				Name:       g.Provider.Name,
				URL:        openapiFilename,
				Type:       arazzo1.SourceDescriptionTypeOpenAPI,
				Extensions: g.Provider.Extensions,
			},
		},
		Components: g.Components,
		Extensions: g.Extensions,
	}

	// Restore Info from Appendices if available
	if g.Provider.Appendices != nil {
		if v, ok := g.Provider.Appendices["info_title"].(string); ok && v != "" {
			arazzo.Info.Title = v
		}
		if v, ok := g.Provider.Appendices["info_version"].(string); ok && v != "" {
			arazzo.Info.Version = v
		}
		if v, ok := g.Provider.Appendices["info_summary"].(string); ok && v != "" {
			arazzo.Info.Summary = v
		}
		if v, ok := g.Provider.Appendices["info_description"].(string); ok && v != "" {
			arazzo.Info.Description = v
		}
	}

	if len(g.Workflows) == 0 {
		return nil, fmt.Errorf("no workflows found in generator config")
	}

	for _, wfSpec := range g.Workflows {
		wf := &arazzo1.Workflow{
			WorkflowId:     wfSpec.WorkflowId,
			Summary:        wfSpec.Summary,
			Description:    wfSpec.Description,
			Inputs:         wfSpec.Inputs,
			DependsOn:      wfSpec.DependsOn,
			Outputs:        wfSpec.Outputs,
			SuccessActions: wfSpec.SuccessActions,
			FailureActions: wfSpec.FailureActions,
			Extensions:     wfSpec.Extensions,
			Steps:          []*arazzo1.Step{},
		}

		// Create steps
		for _, op := range wfSpec.Steps {
			step := &arazzo1.Step{
				StepId:          op.Name,
				Description:     op.Description,
				SuccessCriteria: op.SuccessCriteria,
				OnSuccess:       op.OnSuccess,
				OnFailure:       op.OnFailure,
				Extensions:      op.Extensions,
				Outputs:         op.Outputs,
			}

			// Handle RequestBody (now map[string]interface{} for HCL compat)
			if len(op.RequestBody) > 0 {
				// Check for "payload" key
				if _, hasPayload := op.RequestBody["payload"]; hasPayload {
					// Convert map to struct via JSON (simplest way to handle conversions)
					b, _ := json.Marshal(op.RequestBody)
					var rb arazzo1.RequestBody
					if err := json.Unmarshal(b, &rb); err == nil {
						step.RequestBody = &rb
					}
				} else {
					// Treat entirely as Payload
					step.RequestBody = &arazzo1.RequestBody{
						Payload: op.RequestBody,
					}
				}
			}

			// Handle Target (Operation vs Workflow)
			if op.WorkflowId != "" {
				step.WorkflowId = op.WorkflowId
			} else if op.OperationId != "" {
				step.OperationId = op.OperationId
			} else if op.OperationPath != "" {
				step.OperationPath = op.OperationPath
			} else {
				// Default fallback
				step.OperationId = "$source." + op.Name
			}

			// Copy Parameters
			step.Parameters = op.Parameters

			// Enrichment: This might modify Parameters, RequestBody, SuccessCriteria
			enrichStepFromOpenAPI(step, g.openapiDoc)

			// Add default success criteria if still missing (fallback)
			if len(step.SuccessCriteria) == 0 {
				step.SuccessCriteria = []*arazzo1.Criterion{
					{
						Condition: "$statusCode == 200",
					},
				}
			}

			wf.Steps = append(wf.Steps, step)
		}
		arazzo.Workflows = append(arazzo.Workflows, wf)
	}

	return arazzo, nil
}

// enrichStepFromOpenAPI looks up the operation in the OpenAPI doc and enriches the step parameters.
func enrichStepFromOpenAPI(step *arazzo1.Step, doc *openapi31.OpenAPI) {
	if step.WorkflowId != "" {
		return // Cannot enrich workflow steps from OpenAPI
	}

	// Find operation
	var op *openapi31.Operation
	// Simple lookup by OperationId
	opID := step.OperationId
	// Remove source prefix if present (e.g., "$source.petId")
	if idx := strings.LastIndex(opID, "."); idx != -1 {
		opID = opID[idx+1:]
	}

	if opID != "" && doc.Paths != nil {
		for _, pathItem := range doc.Paths.Paths {
			ops := []*openapi31.Operation{pathItem.Get, pathItem.Put, pathItem.Post, pathItem.Delete, pathItem.Options, pathItem.Head, pathItem.Patch, pathItem.Trace}
			for _, o := range ops {
				if o != nil && o.OperationID == opID {
					op = o
					break
				}
			}
			if op != nil {
				break
			}
		}
	} else if step.OperationPath != "" {
		// Attempt resolution by path
		op = resolveOperationByPath(doc, step.OperationPath)
	}

	if op == nil {
		return
	}

	// Enrichment Logic 1: Auto-fill 'in' for parameters and Auto-include required parameters
	// First, normalize existing parameters and collect names
	existingParams := make(map[string]bool)
	var newParams []interface{}

	for _, pFunc := range step.Parameters {
		// Handle string requests (e.g. "X-Trace-Id")
		if name, ok := pFunc.(string); ok {
			// Find this param in OpenAPI
			found := false
			for _, oasP := range op.Parameters {
				if oasP.Name == name {
					param := &arazzo1.Parameter{
						Name:  oasP.Name,
						In:    arazzo1.ParameterIn(oasP.In),
						Value: "$inputs." + oasP.Name, // Default value
					}
					newParams = append(newParams, param)
					existingParams[name] = true
					found = true
					break
				}
			}
			if !found {
				// If not found in OpenAPI, keep it as is (maybe user knows better or it's extra)
				// But Arazzo Step.Parameters expects *Parameter or map, not string.
				// However, if we are generating, we likely want to resolve it now.
				// If we can't resolve it, we should probably warn or skip.
				// For now, let's convert it to a basic Parameter to avoid type issues later
				param := &arazzo1.Parameter{
					Name:  name,
					Value: "$inputs." + name,
				}
				newParams = append(newParams, param)
				existingParams[name] = true
			}
			continue
		}

		// Handle existing maps/structs
		var name string
		if pMap, ok := pFunc.(map[string]interface{}); ok {
			name, _ = pMap["name"].(string)
			inVal, _ := pMap["in"].(string)
			if name != "" && inVal == "" {
				for _, oasP := range op.Parameters {
					if oasP.Name == name {
						pMap["in"] = oasP.In
						break
					}
				}
			}
			newParams = append(newParams, pMap)
		} else if pStruct, ok := pFunc.(*arazzo1.Parameter); ok {
			name = pStruct.Name
			enrichParameterStruct(pStruct, op)
			newParams = append(newParams, pStruct)
		} else {
			// Unknown type, keep it
			newParams = append(newParams, pFunc)
		}

		if name != "" {
			existingParams[name] = true
		}
	}

	// Second, Auto-include Mandatory Parameters from OpenAPI
	for _, oasP := range op.Parameters {
		if _, exists := existingParams[oasP.Name]; exists {
			continue
		}
		// Logic: Include if Required is true AND NOT Deprecated
		// If Deprecated is true, we skip even if Required (unless user explicitly requested it above)
		if oasP.Required && !oasP.Deprecated {
			param := &arazzo1.Parameter{
				Name:  oasP.Name,
				In:    arazzo1.ParameterIn(oasP.In),
				Value: "$inputs." + oasP.Name,
			}
			newParams = append(newParams, param)
		}
	}

	step.Parameters = newParams

	// Enrichment Logic 2: Security Parameters

	// Enrichment Logic 2: Security Parameters
	if len(op.Security) > 0 && doc.Components != nil && doc.Components.SecuritySchemes != nil {
		// Just take the first requirement set for now
		req := op.Security[0]
		for name := range req {
			if schemeRef, ok := doc.Components.SecuritySchemes[name]; ok {
				// schemeRef might be a reference or value. Assuming value usage simplified for now as generator is mostly reader
				// Actually openapi31.SecuritySchemes is map[string]*SecurityScheme|Reference
				// We need to resolve it. But typically it's direct in components.
				// In tabilet/oas/openapi31, SecurityScheme is struct.
				if schemeRef.Type == "apiKey" {
					// Add parameter
					param := arazzo1.Parameter{
						Name:  schemeRef.Name,
						In:    arazzo1.ParameterIn(schemeRef.In),
						Value: "$inputs." + name, // Heuristic default
					}
					// Only add if not present
					if !parameterExists(step.Parameters, param.Name) {
						step.Parameters = append(step.Parameters, &param)
					}
				} else if schemeRef.Type == "http" {
					headerName := "Authorization"
					if !parameterExists(step.Parameters, headerName) {
						param := arazzo1.Parameter{
							Name:  headerName,
							In:    arazzo1.ParameterInHeader, // Authorization is always header
							Value: "$inputs." + name,
						}
						step.Parameters = append(step.Parameters, &param)
					}
				}
			}
		}
	}

	// Enrichment Logic 3: Dynamic Success Criteria
	if len(step.SuccessCriteria) == 0 && op.Responses != nil && len(op.Responses.StatusCode) > 0 {
		for code := range op.Responses.StatusCode {
			// Check for 2xx codes strings
			if strings.HasPrefix(code, "2") {
				step.SuccessCriteria = append(step.SuccessCriteria, &arazzo1.Criterion{
					Condition: fmt.Sprintf("$statusCode == %s", code),
				})
			}
		}
	}

	// Enrichment Logic 4: Request Body Content-Type and Payload
	if op.RequestBody != nil && len(op.RequestBody.Content) > 0 {
		if step.RequestBody == nil {
			step.RequestBody = &arazzo1.RequestBody{}
		}

		if step.RequestBody.ContentType == "" {
			// Pick first content type
			for ct, mediaType := range op.RequestBody.Content {
				step.RequestBody.ContentType = ct

				// Payload Scaffolding
				if step.RequestBody.Payload == nil {
					if mediaType.Example != nil {
						step.RequestBody.Payload = mediaType.Example
					} else if len(mediaType.Examples) > 0 {
						// Pick first example
						for _, ex := range mediaType.Examples {
							step.RequestBody.Payload = ex.Value
							break
						}
					}
				}
				break
			}
		}
	}
}

func enrichParameterStruct(p *arazzo1.Parameter, op *openapi31.Operation) {
	if p.Name != "" && p.In == "" {
		for _, oasP := range op.Parameters {
			if oasP.Name == p.Name {
				p.In = arazzo1.ParameterIn(oasP.In)
				break
			}
		}
	}
}

func parameterExists(params []any, name string) bool {
	for _, p := range params {
		if pMap, ok := p.(map[string]interface{}); ok {
			if n, _ := pMap["name"].(string); n == name {
				return true
			}
		}
		if pStruct, ok := p.(*arazzo1.Parameter); ok {
			if pStruct.Name == name {
				return true
			}
		}
	}
	return false
}

// resolveOperationByPath resolves a JSON Pointer-like operation path (e.g. #/paths/~1users/get)
func resolveOperationByPath(doc *openapi31.OpenAPI, path string) *openapi31.Operation {
	// Strip source prefix if present (e.g., "$source#/paths...")
	if idx := strings.LastIndex(path, "#"); idx != -1 {
		path = path[idx+1:]
	}

	// Expecting /paths/{path_to_item}/{method}
	parts := strings.Split(path, "/")
	if len(parts) < 4 || parts[1] != "paths" {
		return nil
	}

	// Unescape JSON pointer tokens: ~1 -> /, ~0 -> ~
	unescape := func(s string) string {
		s = strings.ReplaceAll(s, "~1", "/")
		s = strings.ReplaceAll(s, "~0", "~")
		return s
	}

	pathKey := unescape(parts[2]) // The path key, e.g. /users
	method := parts[3]            // The method, e.g. get

	if doc.Paths == nil {
		return nil
	}

	for key, pathItem := range doc.Paths.Paths {
		if key == pathKey {
			switch strings.ToLower(method) {
			case "get":
				return pathItem.Get
			case "put":
				return pathItem.Put
			case "post":
				return pathItem.Post
			case "delete":
				return pathItem.Delete
			case "options":
				return pathItem.Options
			case "head":
				return pathItem.Head
			case "patch":
				return pathItem.Patch
			case "trace":
				return pathItem.Trace
			}
		}
	}

	return nil
}
