package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type flagEntry struct {
	field string
	value any
}

func parseFlags(args []string) (map[string]any, error) {
	root := make(map[string]any)
	entries, err := parseFlagEntries(args)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if err := setNested(root, strings.Split(entry.field, "."), entry.value); err != nil {
			return nil, err
		}
	}
	return root, nil
}

func parseFlagValues(args []string) (map[string]any, error) {
	values := make(map[string]any)
	entries, err := parseFlagEntries(args)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		values[entry.field] = entry.value
	}
	return values, nil
}

func parseFlagEntries(args []string) ([]flagEntry, error) {
	var entries []flagEntry
	for i := 0; i < len(args); {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") || len(arg) == 2 {
			return nil, fmt.Errorf("expected --field VALUE, got %q", arg)
		}
		field, raw, hasValue := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		if !hasValue {
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("missing value for --%s", field)
			}
			raw = args[i]
		}
		if field == "" {
			return nil, errors.New("field name cannot be empty")
		}
		entries = append(entries, flagEntry{field: field, value: parseFlagValue(raw)})
		i++
	}
	return entries, nil
}

func parseFlagValue(raw string) any {
	var value any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err == nil {
		var trailing any
		if decoder.Decode(&trailing) == io.EOF {
			return value
		}
	}
	return raw
}

func setNested(root map[string]any, path []string, value any) error {
	if len(path) == 0 {
		return errors.New("field name cannot be empty")
	}
	for _, part := range path {
		if part == "" {
			return errors.New("field path cannot contain empty segments")
		}
	}
	current := root
	for _, part := range path[:len(path)-1] {
		child, exists := current[part]
		if !exists {
			created := make(map[string]any)
			current[part] = created
			current = created
			continue
		}
		object, ok := child.(map[string]any)
		if !ok {
			return fmt.Errorf("conflicting field path at %q", part)
		}
		current = object
	}
	leaf := path[len(path)-1]
	if existing, ok := current[leaf]; ok {
		if _, object := existing.(map[string]any); object {
			if _, replacementObject := value.(map[string]any); !replacementObject {
				return fmt.Errorf("conflicting values for field %q", leaf)
			}
		}
	}
	current[leaf] = value
	return nil
}

func mergeMaps(destination, changes map[string]any) {
	for key, change := range changes {
		changeObject, ok := change.(map[string]any)
		if !ok {
			destination[key] = change
			continue
		}
		destinationObject, ok := destination[key].(map[string]any)
		if !ok {
			destinationObject = make(map[string]any)
			destination[key] = destinationObject
		}
		mergeMaps(destinationObject, changeObject)
	}
}

func queryValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}
