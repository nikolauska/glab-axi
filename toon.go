package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const previewLimit = 1000

func renderList(label string, items []any, fields []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s[%d]{%s}:\n", label, len(items), strings.Join(fields, ","))
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		row := make([]string, len(fields))
		for i, field := range fields {
			row[i] = scalar(fieldValue(item, field), ',')
		}
		if len(row) > 0 {
			b.WriteString("  " + strings.Join(row, ",") + "\n")
		}
	}
	return b.String()
}

func renderObject(label string, item map[string]any) string {
	return label + ":\n" + indent(encodeMap(item), 2) + "\n"
}

func selectFields(item map[string]any, fields []string, full bool) (map[string]any, bool) {
	out, truncated := map[string]any{}, false
	for _, field := range fields {
		value := fieldValue(item, field)
		if text, ok := value.(string); ok && !full && len([]rune(text)) > previewLimit {
			r := []rune(text)
			value = string(r[:previewLimit]) + fmt.Sprintf("... (truncated, %d chars total)", len(r))
			truncated = true
		}
		out[field] = value
	}
	return out, truncated
}

func fieldValue(item map[string]any, path string) any {
	var value any = item
	for _, part := range strings.Split(path, ".") {
		m, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = m[part]
	}
	if list, ok := value.([]any); ok {
		parts := make([]string, 0, len(list))
		for _, v := range list {
			if m, ok := v.(map[string]any); ok {
				if name, ok := m["username"]; ok {
					v = name
				}
			}
			parts = append(parts, fmt.Sprint(v))
		}
		return strings.Join(parts, "|")
	}
	return value
}

func encodeTOON(value any) string {
	switch v := value.(type) {
	case map[string]any:
		return encodeMap(v)
	case []any:
		var b strings.Builder
		fmt.Fprintf(&b, "items[%d]:", len(v))
		for _, item := range v {
			b.WriteString("\n  - " + strings.ReplaceAll(encodeTOON(item), "\n", "\n    "))
		}
		return b.String()
	default:
		return scalar(v, 0)
	}
}

func encodeMap(v map[string]any) string {
	keys := make([]string, 0, len(v))
	for key := range v {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var lines []string
	for _, key := range keys {
		switch value := v[key].(type) {
		case map[string]any:
			lines = append(lines, quoteKey(key)+":\n"+indent(encodeMap(value), 2))
		case []any:
			lines = append(lines, encodeArrayField(key, value))
		default:
			lines = append(lines, quoteKey(key)+": "+scalar(value, 0))
		}
	}
	return strings.Join(lines, "\n")
}

func encodeArrayField(key string, values []any) string {
	header := fmt.Sprintf("%s[%d]:", quoteKey(key), len(values))
	if len(values) == 0 {
		return header
	}
	primitive := true
	for _, value := range values {
		switch value.(type) {
		case map[string]any, []any:
			primitive = false
		}
	}
	if primitive {
		parts := make([]string, len(values))
		for i, value := range values {
			parts[i] = scalar(value, ',')
		}
		return header + " " + strings.Join(parts, ",")
	}
	var lines []string
	for _, value := range values {
		switch item := value.(type) {
		case map[string]any:
			encoded := encodeMap(item)
			parts := strings.SplitN(encoded, "\n", 2)
			line := "  - " + parts[0]
			if len(parts) == 2 {
				line += "\n" + indent(parts[1], 4)
			}
			lines = append(lines, line)
		case []any:
			lines = append(lines, "  - items"+strings.TrimPrefix(encodeArrayField("", item), "["))
		default:
			lines = append(lines, "  - "+scalar(item, 0))
		}
	}
	return header + "\n" + strings.Join(lines, "\n")
}

func scalar(value any, delimiter byte) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case string:
		return quoteFor(v, delimiter)
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = scalar(item, ',')
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		return quoteFor(fmt.Sprint(v), delimiter)
	}
}

func quote(s string) string { return quoteFor(s, 0) }
func quoteFor(s string, delimiter byte) string {
	needs := s == "" || strings.TrimSpace(s) != s || s == "true" || s == "false" || s == "null" || strings.ContainsAny(s, "\n\r\t:\"\\[]{}") || (delimiter != 0 && strings.IndexByte(s, delimiter) >= 0)
	if !needs {
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return s
		}
	}
	return strconv.Quote(s)
}
func quoteKey(s string) string {
	if strings.ContainsAny(s, ":\n\r\t[]{}\"") || s == "" {
		return strconv.Quote(s)
	}
	return s
}
func indent(s string, n int) string {
	pad := strings.Repeat(" ", n)
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}
