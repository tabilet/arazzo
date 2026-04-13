// Package convert provides functions to convert Arazzo documents between JSON and HCL formats.
package convert

import (
	"encoding/json"
	"strings"

	"github.com/tabilet/arazzo/arazzo1"
	"github.com/genelet/horizon/dethcl"
)

const hclDollarKeyPrefix = "__dollar__"

var legacyDollarKeys = map[string]struct{}{
	"$ref":           {},
	"$id":            {},
	"$schema":        {},
	"$defs":          {},
	"$comment":       {},
	"$vocabulary":    {},
	"$anchor":        {},
	"$dynamicRef":    {},
	"$dynamicAnchor": {},
}

func toHCLKey(key string) string {
	if !strings.HasPrefix(key, "$") {
		return key
	}
	if _, ok := legacyDollarKeys[key]; ok {
		return "_" + key[1:]
	}
	return hclDollarKeyPrefix + key[1:]
}

func fromHCLKey(key string) string {
	if strings.HasPrefix(key, hclDollarKeyPrefix) {
		return "$" + key[len(hclDollarKeyPrefix):]
	}
	if strings.HasPrefix(key, "_") {
		candidate := "$" + key[1:]
		if _, ok := legacyDollarKeys[candidate]; ok {
			return candidate
		}
	}
	return key
}

// transformValue recursively transforms values for HCL compatibility.
// When toHCL is true:
//   - Converts $-prefixed keys to HCL-safe identifiers
//   - Escapes newlines in strings (HCL quoted strings cannot span multiple lines)
//
// When toHCL is false:
//   - Converts HCL-safe identifiers back to $-prefixed keys
//   - Unescapes newlines in strings
func transformValue(v any, toHCL bool) any {
	switch val := v.(type) {
	case string:
		if toHCL {
			// Escape newlines and quotes for HCL compatibility
			return escapeForHCL(val)
		}
		// Unescape newlines and quotes when converting back from HCL
		return unescapeFromHCL(val)
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			newKey := k
			if toHCL {
				newKey = toHCLKey(k)
			} else {
				newKey = fromHCLKey(k)
			}
			result[newKey] = transformValue(v, toHCL)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = transformValue(item, toHCL)
		}
		return result
	default:
		return v
	}
}

// escapeForHCL escapes special characters in strings for HCL compatibility.
// HCL quoted strings cannot span multiple lines, so newlines are escaped as \n.
// Internal double quotes are escaped as \".
func escapeForHCL(s string) string {
	// First protect already-escaped sequences
	s = strings.ReplaceAll(s, "\\n", "\x00ESCAPED_N\x00")
	s = strings.ReplaceAll(s, "\\\"", "\x00ESCAPED_Q\x00")

	// Escape newlines and quotes
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\"", "\\\"")

	// Restore double-escaped sequences
	s = strings.ReplaceAll(s, "\x00ESCAPED_N\x00", "\\\\n")
	s = strings.ReplaceAll(s, "\x00ESCAPED_Q\x00", "\\\\\"")
	return s
}

// escapeNewlines replaces actual newlines with escaped \n sequences
// so that HCL quoted strings remain on a single line.
// This is used for string fields that go through standard JSON marshaling.
func escapeNewlines(s string) string {
	// Replace actual newlines with escaped sequence
	// We use a placeholder first to avoid double-escaping existing \n
	s = strings.ReplaceAll(s, "\\n", "\x00ESCAPED_N\x00")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\x00ESCAPED_N\x00", "\\\\n")
	return s
}

// unescapeNewlines converts escaped \n sequences back to actual newlines.
func unescapeNewlines(s string) string {
	// First protect already escaped backslashes followed by n
	s = strings.ReplaceAll(s, "\\\\n", "\x00ESCAPED_N\x00")
	// Then convert \n to actual newline
	s = strings.ReplaceAll(s, "\\n", "\n")
	// Restore escaped sequences
	s = strings.ReplaceAll(s, "\x00ESCAPED_N\x00", "\\n")
	return s
}

// unescapeFromHCL unescapes special characters from HCL strings.
func unescapeFromHCL(s string) string {
	// First protect double-escaped sequences
	s = strings.ReplaceAll(s, "\\\\n", "\x00ESCAPED_N\x00")
	s = strings.ReplaceAll(s, "\\\\\"", "\x00ESCAPED_Q\x00")

	// Unescape newlines and quotes
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\\"", "\"")

	// Restore double-escaped as single-escaped
	s = strings.ReplaceAll(s, "\x00ESCAPED_N\x00", "\\n")
	s = strings.ReplaceAll(s, "\x00ESCAPED_Q\x00", "\\\"")
	return s
}

// transformArazzoForHCL transforms an Arazzo document's dynamic fields ($ref -> _ref) for HCL compatibility.
// It also escapes newlines in string fields since HCL quoted strings cannot span multiple lines.
func transformArazzoForHCL(doc *arazzo1.Arazzo) {
	// Transform string fields with potential newlines
	if doc.Info != nil {
		doc.Info.Description = escapeNewlines(doc.Info.Description)
		doc.Info.Summary = escapeNewlines(doc.Info.Summary)
	}
	for _, wf := range doc.Workflows {
		wf.Description = escapeNewlines(wf.Description)
		wf.Summary = escapeNewlines(wf.Summary)
		// Transform workflow inputs
		if wf.Inputs != nil {
			wf.Inputs = transformValue(wf.Inputs, true)
		}
		for _, step := range wf.Steps {
			step.Description = escapeNewlines(step.Description)
			// Transform step parameters ([]any may contain maps with $ref keys and numeric values)
			if step.Parameters != nil {
				for i, param := range step.Parameters {
					step.Parameters[i] = transformValue(param, true)
				}
			}
			if step.RequestBody != nil {
				// Transform any-typed Payload and Replacements
				if step.RequestBody.Payload != nil {
					step.RequestBody.Payload = transformValue(step.RequestBody.Payload, true)
				}
			}
		}
	}
	// Transform component inputs
	if doc.Components != nil && doc.Components.Inputs != nil {
		for k, v := range doc.Components.Inputs {
			doc.Components.Inputs[k] = transformValue(v, true)
		}
	}
}

// transformArazzoFromHCL transforms an Arazzo document's dynamic fields (_ref -> $ref) back from HCL.
// It also unescapes newlines in string fields.
func transformArazzoFromHCL(doc *arazzo1.Arazzo) {
	// Transform string fields with escaped newlines
	if doc.Info != nil {
		doc.Info.Description = unescapeNewlines(doc.Info.Description)
		doc.Info.Summary = unescapeNewlines(doc.Info.Summary)
	}
	for _, wf := range doc.Workflows {
		wf.Description = unescapeNewlines(wf.Description)
		wf.Summary = unescapeNewlines(wf.Summary)
		// Transform workflow inputs
		if wf.Inputs != nil {
			wf.Inputs = transformValue(wf.Inputs, false)
		}
		for _, step := range wf.Steps {
			step.Description = unescapeNewlines(step.Description)
			// Transform step parameters ([]any may contain maps with _ref keys)
			if step.Parameters != nil {
				for i, param := range step.Parameters {
					step.Parameters[i] = transformValue(param, false)
				}
			}
			if step.RequestBody != nil {
				// Transform any-typed Payload and Replacements
				if step.RequestBody.Payload != nil {
					step.RequestBody.Payload = transformValue(step.RequestBody.Payload, false)
				}
			}
		}
	}
	// Transform component inputs
	if doc.Components != nil && doc.Components.Inputs != nil {
		for k, v := range doc.Components.Inputs {
			doc.Components.Inputs[k] = transformValue(v, false)
		}
	}
}

// JSONToHCL converts an Arazzo document from JSON format to HCL format.
// It first unmarshals the JSON into an Arazzo struct, then marshals it to HCL.
// JSON Schema keys like $ref are transformed to _ref for HCL compatibility.
func JSONToHCL(jsonData []byte) ([]byte, error) {
	var doc arazzo1.Arazzo
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		return nil, err
	}
	transformArazzoForHCL(&doc)
	return dethcl.Marshal(&doc)
}

// HCLToJSON converts an Arazzo document from HCL format to JSON format.
// It first unmarshals the HCL into an Arazzo struct, then marshals it to JSON.
// HCL keys like _ref are transformed back to $ref for JSON compatibility.
func HCLToJSON(hclData []byte) ([]byte, error) {
	var doc arazzo1.Arazzo
	if err := dethcl.Unmarshal(hclData, &doc); err != nil {
		return nil, err
	}
	transformArazzoFromHCL(&doc)
	return json.Marshal(&doc)
}

// HCLToJSONIndent converts an Arazzo document from HCL format to indented JSON format.
// HCL keys like _ref are transformed back to $ref for JSON compatibility.
func HCLToJSONIndent(hclData []byte, prefix, indent string) ([]byte, error) {
	var doc arazzo1.Arazzo
	if err := dethcl.Unmarshal(hclData, &doc); err != nil {
		return nil, err
	}
	transformArazzoFromHCL(&doc)
	return json.MarshalIndent(&doc, prefix, indent)
}

// MarshalHCL marshals an Arazzo document to HCL format.
// JSON Schema keys like $ref are transformed to _ref for HCL compatibility.
// Note: This function modifies the document in place. If you need to preserve
// the original, make a copy before calling this function.
func MarshalHCL(doc *arazzo1.Arazzo) ([]byte, error) {
	transformArazzoForHCL(doc)
	return dethcl.Marshal(doc)
}

// UnmarshalHCL unmarshals HCL data into an Arazzo document.
// HCL keys like _ref are transformed back to $ref for JSON compatibility.
func UnmarshalHCL(hclData []byte, doc *arazzo1.Arazzo) error {
	if err := dethcl.Unmarshal(hclData, doc); err != nil {
		return err
	}
	transformArazzoFromHCL(doc)
	return nil
}

// MarshalJSON marshals an Arazzo document to JSON format.
func MarshalJSON(doc *arazzo1.Arazzo) ([]byte, error) {
	return json.Marshal(doc)
}

// MarshalJSONIndent marshals an Arazzo document to indented JSON format.
func MarshalJSONIndent(doc *arazzo1.Arazzo, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(doc, prefix, indent)
}

// UnmarshalJSON unmarshals JSON data into an Arazzo document.
func UnmarshalJSON(jsonData []byte, doc *arazzo1.Arazzo) error {
	return json.Unmarshal(jsonData, doc)
}
