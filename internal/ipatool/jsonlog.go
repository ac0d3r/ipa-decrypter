package ipatool

import (
	"encoding/json"
	"slices"
	"strings"
)

// ParseLogObjects scans zerolog JSON lines from ipatool --format json output.
func ParseLogObjects(output string) []map[string]any {
	var out []map[string]any
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func LastSuccessLog(output string) (map[string]any, bool) {
	objs := ParseLogObjects(output)
	for i := range slices.Backward(objs) {
		if success, ok := objs[i]["success"].(bool); ok && success {
			return objs[i], true
		}
	}
	return nil, false
}

func StringSliceField(m map[string]any, key string) []string {
	raw, ok := m[key]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	default:
		return nil
	}
}

func StringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// FailureMessage extracts the most recent error from ipatool JSON logs.
func FailureMessage(output string) string {
	objs := ParseLogObjects(output)
	for i := range slices.Backward(objs) {
		obj := objs[i]
		if success, ok := obj["success"].(bool); ok && !success {
			if e := StringField(obj, "error"); e != "" {
				return e
			}
		}
		if level, _ := obj["level"].(string); level == "error" {
			if e := StringField(obj, "message"); e != "" {
				return e
			}
			if e := StringField(obj, "error"); e != "" {
				return e
			}
		}
	}
	return ""
}
