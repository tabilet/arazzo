# Arazzo Go Parser

A Go library for parsing, validating, and generating [Arazzo 1.0](https://spec.openapis.org/arazzo/v1.0.0) documents. Arazzo is an OpenAPI Initiative specification for describing workflows that span multiple APIs.

[![GoDoc](https://godoc.org/github.com/tabilet/arazzo?status.svg)](https://godoc.org/github.com/tabilet/arazzo)

## Installation

```bash
go get github.com/tabilet/arazzo
```

## Features

- Full support for Arazzo 1.0.x specification
- Marshal/Unmarshal JSON with proper round-trip preservation
- **HCL format support** - Convert between JSON and HCL representations
- Specification extensions (`x-*`) support on all objects
- Comprehensive validation with detailed error paths
- Type-safe constants for enum values

## Quick Start

### Parsing an Arazzo Document

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/tabilet/arazzo/arazzo1"
)

func main() {
    // Read the Arazzo document
    data, err := os.ReadFile("workflow.arazzo.json")
    if err != nil {
        log.Fatal(err)
    }

    // Parse it
    var doc arazzo1.Arazzo
    if err := json.Unmarshal(data, &doc); err != nil {
        log.Fatal(err)
    }

    // Access the parsed data
    fmt.Printf("Title: %s\n", doc.Info.Title)
    fmt.Printf("Version: %s\n", doc.Info.Version)
    fmt.Printf("Workflows: %d\n", len(doc.Workflows))

    for _, wf := range doc.Workflows {
        fmt.Printf("  - %s: %d steps\n", wf.WorkflowId, len(wf.Steps))
    }
}
```

### Validating a Document

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/tabilet/arazzo/arazzo1"
)

func main() {
    data := []byte(`{
        "arazzo": "1.0.0",
        "info": {"title": "My Workflow", "version": "1.0.0"},
        "sourceDescriptions": [
            {"name": "petstore", "url": "./openapi.json", "type": "openapi"}
        ],
        "workflows": [
            {
                "workflowId": "get-pet",
                "steps": [
                    {"stepId": "fetch", "operationId": "getPetById"}
                ]
            }
        ]
    }`)

    var doc arazzo1.Arazzo
    if err := json.Unmarshal(data, &doc); err != nil {
        log.Fatal(err)
    }

    // Validate the document
    result := doc.Validate()
    if !result.Valid() {
        fmt.Println("Validation errors:")
        for _, err := range result.Errors {
            fmt.Printf("  %s: %s\n", err.Path, err.Message)
        }
        return
    }

    fmt.Println("Document is valid!")
}
```

### Creating a Document Programmatically

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/tabilet/arazzo/arazzo1"
)

func main() {
    // Build the document
    doc := &arazzo1.Arazzo{
        Arazzo: "1.0.0",
        Info: &arazzo1.Info{
            Title:       "Pet Store Workflow",
            Summary:     "Workflows for managing pets",
            Description: "A collection of workflows for the Pet Store API",
            Version:     "1.0.0",
        },
        SourceDescriptions: []*arazzo1.SourceDescription{
            {
                Name: "petstore",
                URL:  "https://petstore3.swagger.io/api/v3/openapi.json",
                Type: arazzo1.SourceDescriptionTypeOpenAPI,
            },
        },
        Workflows: []*arazzo1.Workflow{
            {
                WorkflowId:  "create-and-get-pet",
                Summary:     "Create a pet and retrieve it",
                Description: "This workflow creates a new pet and then fetches it by ID",
                Steps: []*arazzo1.Step{
                    {
                        StepId:      "create-pet",
                        OperationId: "addPet",
                        RequestBody: &arazzo1.RequestBody{
                            ContentType: "application/json",
                            Payload: map[string]any{
                                "name":   "$inputs.petName",
                                "status": "available",
                            },
                        },
                        SuccessCriteria: []*arazzo1.Criterion{
                            {Condition: "$statusCode == 200"},
                        },
                        Outputs: map[string]string{
                            "petId": "$response.body.id",
                        },
                    },
                    {
                        StepId:      "get-pet",
                        OperationId: "getPetById",
                        Parameters: []any{
                            &arazzo1.Parameter{
                                Name:  "petId",
                                In:    arazzo1.ParameterInPath,
                                Value: "$steps.create-pet.outputs.petId",
                            },
                        },
                    },
                },
                Outputs: map[string]string{
                    "pet": "$steps.get-pet.outputs.response",
                },
            },
        },
    }

    // Validate before serializing
    if result := doc.Validate(); !result.Valid() {
        log.Fatalf("Invalid document: %s", result.Error())
    }

    // Marshal to JSON
    output, err := json.MarshalIndent(doc, "", "  ")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(string(output))
}
```

## Type Reference

### Main Types

| Type | Description |
|------|-------------|
| `Arazzo` | Root document object |
| `Info` | Metadata about the Arazzo description |
| `SourceDescription` | Reference to an OpenAPI or Arazzo document |
| `Workflow` | A workflow with steps |
| `Step` | A single step in a workflow |
| `Parameter` | A parameter for operations or workflows |
| `RequestBody` | Request body for an operation |
| `PayloadReplacement` | Dynamic value replacement in payloads |
| `Criterion` | Success/failure assertion |
| `CriterionExpressionType` | Expression type with version |
| `SuccessAction` | Action on step success |
| `FailureAction` | Action on step failure |
| `ReusableObject` | Reference to a reusable component |
| `Components` | Container for reusable objects |

### Union Types

| Type | Description |
|------|-------------|
| `ParameterOrReusable` | Either a Parameter or ReusableObject |
| `SuccessActionOrReusable` | Either a SuccessAction or ReusableObject |
| `FailureActionOrReusable` | Either a FailureAction or ReusableObject |

### Enum Constants

```go
// Source Description Types
arazzo1.SourceDescriptionTypeArazzo  // "arazzo"
arazzo1.SourceDescriptionTypeOpenAPI // "openapi"

// Parameter Locations
arazzo1.ParameterInPath   // "path"
arazzo1.ParameterInQuery  // "query"
arazzo1.ParameterInHeader // "header"
arazzo1.ParameterInCookie // "cookie"

// Criterion Types
arazzo1.CriterionTypeSimple   // "simple"
arazzo1.CriterionTypeRegex    // "regex"
arazzo1.CriterionTypeJSONPath // "jsonpath"
arazzo1.CriterionTypeXPath    // "xpath"

// Success Action Types
arazzo1.SuccessActionTypeEnd  // "end"
arazzo1.SuccessActionTypeGoto // "goto"

// Failure Action Types
arazzo1.FailureActionTypeEnd   // "end"
arazzo1.FailureActionTypeGoto  // "goto"
arazzo1.FailureActionTypeRetry // "retry"
```

## HCL Format Support

The `convert` package provides functions to convert Arazzo documents between JSON and HCL formats using [genelet/horizon](https://github.com/genelet/horizon).

### Converting JSON to HCL

```go
package main

import (
    "fmt"
    "log"

    "github.com/tabilet/arazzo/convert"
)

func main() {
    jsonData := []byte(`{
        "arazzo": "1.0.0",
        "info": {"title": "My Workflow", "version": "1.0.0"},
        "sourceDescriptions": [
            {"name": "petstore", "url": "./openapi.json", "type": "openapi"}
        ],
        "workflows": [
            {
                "workflowId": "get-pet",
                "steps": [{"stepId": "fetch", "operationId": "getPetById"}]
            }
        ]
    }`)

    hclData, err := convert.JSONToHCL(jsonData)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(string(hclData))
}
```

Output:
```hcl
arazzo = "1.0.0"

info {
  title   = "My Workflow"
  version = "1.0.0"
}

sourceDescription "petstore" {
  url  = "./openapi.json"
  type = "openapi"
}

workflow "get-pet" {
  step "fetch" {
    operationId = "getPetById"
  }
}
```

### Converting HCL to JSON

```go
package main

import (
    "fmt"
    "log"

    "github.com/tabilet/arazzo/convert"
)

func main() {
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

    jsonData, err := convert.HCLToJSONIndent(hclData, "", "  ")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(string(jsonData))
}
```

### Direct Marshal/Unmarshal with HCL

```go
package main

import (
    "fmt"
    "log"

    "github.com/tabilet/arazzo/arazzo1"
    "github.com/tabilet/arazzo/convert"
)

func main() {
    // Create a document
    doc := &arazzo1.Arazzo{
        Arazzo: "1.0.0",
        Info: &arazzo1.Info{
            Title:   "My API",
            Version: "1.0.0",
        },
        SourceDescriptions: []*arazzo1.SourceDescription{
            {Name: "api", URL: "./api.json"},
        },
        Workflows: []*arazzo1.Workflow{
            {
                WorkflowId: "test",
                Steps: []*arazzo1.Step{
                    {StepId: "s1", OperationId: "op1"},
                },
            },
        },
    }

    // Marshal to HCL
    hclData, err := convert.MarshalHCL(doc)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(hclData))

    // Unmarshal from HCL
    var doc2 arazzo1.Arazzo
    if err := convert.UnmarshalHCL(hclData, &doc2); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Parsed: %s\n", doc2.Info.Title)
}
```

### Convert Package Functions

| Function | Description |
|----------|-------------|
| `JSONToHCL(jsonData []byte)` | Convert JSON to HCL |
| `HCLToJSON(hclData []byte)` | Convert HCL to JSON |
| `HCLToJSONIndent(hclData []byte, prefix, indent string)` | Convert HCL to indented JSON |
| `MarshalHCL(doc *arazzo1.Arazzo)` | Marshal Arazzo document to HCL |
| `UnmarshalHCL(hclData []byte, doc *arazzo1.Arazzo)` | Unmarshal HCL to Arazzo document |
| `MarshalJSON(doc *arazzo1.Arazzo)` | Marshal Arazzo document to JSON |
| `MarshalJSONIndent(doc *arazzo1.Arazzo, prefix, indent string)` | Marshal to indented JSON |
| `UnmarshalJSON(jsonData []byte, doc *arazzo1.Arazzo)` | Unmarshal JSON to Arazzo document |

### HCL Conversion Notes

**JSON Schema `$ref` Handling**: JSON Schema keys starting with `$` (like `$ref`, `$id`, `$schema`) are automatically transformed to use `_` prefix (e.g., `_ref`) when converting to HCL, since `$` is not valid in HCL identifiers. The transformation is reversed when converting back to JSON.

**String Escaping**: Multi-line strings and strings containing embedded quotes are automatically escaped when converting to HCL and unescaped when converting back. Newlines become `\n` sequences in HCL output.

**Primitive Values in `any` Fields**: Primitive values (strings, numbers, booleans) in dynamically-typed fields (like `RequestBody.Payload` and `Parameter.Value`) are correctly rendered as HCL attributes and properly round-trip through conversions. This includes numeric values in component parameters and step parameter arrays.

**Full Round-Trip Support**: All Arazzo documents round-trip correctly through HCL, including complex cases with numeric parameter values and nested structures. Both JSON and HCL formats maintain full fidelity.

## Arazzo Generator

The `generator` package allows you to automatically create Arazzo specifications from existing OpenAPI 3.0/3.1 documents. It uses a configuration file to define workflows and steps, while leveraging the OpenAPI definition to enrich the output with high-fidelity details.

### Features

-   **Intelligent Enrichment**: Automatically populates `parameters`, `security` headers, `successCriteria`, and `requestBody` payloads from the OpenAPI definition.
-   **Operation Resolution**: Supports referencing operations by `operationId` or JSON Pointer `operationPath` (e.g., `#/paths/~1users/get`).
-   **Multi-Workflow Support**: Define multiple workflows in a single configuration.
-   **Flexible Configuration**: Supports Generator configuration in YAML, JSON, or HCL formats.
-   **Auto-generation**: Simple string list layout for parameters (e.g. `parameters: ["id", "trace_id"]`) to automatically fetch definitions from OpenAPI.

### Usage

#### 1. Create a Generator Configuration (generator.yaml)

```yaml
provider:
  name: petstore
  server_url: http://petstore.swagger.io/v2
  extensions:
    x-env: production

workflows:
  - workflow_id: create-and-get-pet
    summary: Create a pet and retrieve it
    steps:
      - name: createPet
        operation_id: addPet
        request_body:
          # Payload will be auto-scaffolded from OpenAPI examples if omitted
          # Keys must match Arazzo spec (camelCase) because they map directly to RequestBody struct
          contentType: application/json
        outputs:
          petId: $response.body.id
      
      - name: getPet
        operation_path: "#/paths/~1pet~1{petId}/get"
        parameters:
          # Option 1: Full definition (Standard Arazzo)
          - name: petId
            value: $steps.createPet.outputs.petId
          # Option 2: Simple String (Auto-lookup from OpenAPI)
          # "trace-id" 

```

#### 2. Generate Arazzo

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/tabilet/arazzo/generator"
)

func main() {
    // Generate Arazzo from OpenAPI and Generator config
    arazzo, err := generator.NewArazzoFromFiles(
        "openapi.yaml",   // Path to OpenAPI definition
        "generator.yaml", // Path to Generator configuration
    )
    if err != nil {
        log.Fatal(err)
    }

    // Output as JSON
    bytes, _ := json.MarshalIndent(arazzo, "", "  ")
    fmt.Println(string(bytes))
}
```

#### 3. Using HCL Configuration (generator.hcl)

```hcl
provider "petstore" {
  server_url = "http://petstore.swagger.io/v2"
}

workflow "create-and-get-pet" {
  summary = "Create a pet and retrieve it"
  
  step "createPet" {
    operation_id = "addPet"
    
    # Request body can be assigned as a map
    request_body = {
      contentType = "application/json"
      # payload = { ... } # Optional explicit payload
    }

    outputs {
      petId = "$response.body.id"
    }
  }

  step "getPet" {
    operation_path = "#/paths/~1pet~1{petId}/get"
    
    parameter {
      name  = "petId"
      value = "$steps.createPet.outputs.petId"
    }
  }
}
```

```go
// Use "hcl" as format argument
arazzo, err := generator.NewArazzoFromFiles("openapi.yaml", "generator.hcl", "hcl")
```

## Validation

The `Validate()` method performs comprehensive validation:

- Required fields presence
- Pattern matching (arazzo version, names)
- Enum value validation
- Mutual exclusivity (e.g., step must have exactly one of operationId/operationPath/workflowId)
- Conditional requirements (e.g., goto action requires stepId or workflowId)
- Component name patterns
- Nested object validation

```go
result := doc.Validate()
if !result.Valid() {
    for _, err := range result.Errors {
        fmt.Printf("%s: %s\n", err.Path, err.Message)
    }
}
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related Projects

- [tabilet/oas](https://github.com/tabilet/oas) - Go parser for OpenAPI 3.0 and 3.1 specifications
- [genelet/horizon](https://github.com/genelet/horizon) - HCL parsing library used for HCL format support
- [Arazzo Specification](https://spec.openapis.org/arazzo/v1.0.0) - Official Arazzo specification
