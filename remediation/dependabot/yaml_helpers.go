package dependabot

import (
	"bytes"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// findMappingValue returns the value node for the given key inside a YAML mapping node.
func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// findLastLine returns the maximum 1-indexed line number among node and all its descendants.
func findLastLine(node *yaml.Node) int {
	last := node.Line
	for _, child := range node.Content {
		if l := findLastLine(child); l > last {
			last = l
		}
	}
	return last
}

// getChildIndent returns the 0-indexed column of the first key in a mapping node.
func getChildIndent(mappingNode *yaml.Node) int {
	if mappingNode.Kind == yaml.MappingNode && len(mappingNode.Content) > 0 {
		return mappingNode.Content[0].Column - 1
	}
	return 0
}

// marshalYAMLValue returns the YAML-safe representation of a scalar string,
// adding quotes when needed (e.g. '*' becomes '\'*\").
func marshalYAMLValue(v string) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		return v
	}
	return strings.TrimRight(string(b), "\n")
}

// insertAfterLine inserts newLines into lines after the given 1-indexed line number.
func insertAfterLine(lines []string, afterLine int, newLines []string) []string {
	idx := afterLine // 1-indexed line N → insert at 0-indexed position N
	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:idx]...)
	result = append(result, newLines...)
	result = append(result, lines[idx:]...)
	return result
}

// replaceScalarOnLine replaces the old value at the given node position with newVal,
// preserving any trailing content (e.g. comments) on the same line.
func replaceScalarOnLine(lines []string, lineIdx int, node *yaml.Node, newVal string) {
	line := lines[lineIdx]
	col := node.Column - 1
	end := col + len(node.Value)
	// When the original value is quoted, node.Column points to the opening quote
	// but node.Value is the unquoted content. Account for the surrounding quotes.
	if node.Style == yaml.DoubleQuotedStyle || node.Style == yaml.SingleQuotedStyle {
		end += 2 // skip both opening and closing quote
		if node.Style == yaml.DoubleQuotedStyle {
			newVal = "\"" + newVal + "\""
		} else {
			newVal = "'" + newVal + "'"
		}
	}
	if end > len(line) {
		end = len(line)
	}
	lines[lineIdx] = line[:col] + newVal + line[end:]
}

// replaceSequence replaces the items of a sequence node in lines with newValues.
// Handles both flow style ([a, b]) and block style (- a\n- b).
// Returns (updated lines, net line count change, whether changed).
func replaceSequence(lines []string, seqNode *yaml.Node, newValues []string, lineOffset int) ([]string, int, bool) {
	if seqNode.Style == yaml.FlowStyle {
		lineIdx := seqNode.Line - 1 + lineOffset
		line := lines[lineIdx]
		col := seqNode.Column - 1
		openIdx := strings.Index(line[col:], "[")
		if openIdx < 0 {
			return lines, 0, false
		}
		actualOpen := col + openIdx
		closeIdx := strings.LastIndex(line, "]")
		if closeIdx <= actualOpen {
			return lines, 0, false
		}
		quotedVals := make([]string, len(newValues))
		for i, v := range newValues {
			quotedVals[i] = marshalYAMLValue(v)
		}
		newSeq := "[" + strings.Join(quotedVals, ", ") + "]"
		oldSeq := line[actualOpen : closeIdx+1]
		if oldSeq == newSeq {
			return lines, 0, false
		}
		result := make([]string, len(lines))
		copy(result, lines)
		result[lineIdx] = line[:actualOpen] + newSeq + line[closeIdx+1:]
		return result, 0, true
	}

	// Block sequence
	if len(seqNode.Content) == 0 {
		return lines, 0, false
	}
	firstItemLine := seqNode.Content[0].Line - 1 + lineOffset
	lastItemLine := findLastLine(seqNode) - 1 + lineOffset
	itemIndent := seqNode.Column - 1 // spaces before the '-'

	newItemLines := make([]string, len(newValues))
	for i, v := range newValues {
		newItemLines[i] = strings.Repeat(" ", itemIndent) + "- " + marshalYAMLValue(v)
	}

	// Check if content is actually different
	oldCount := lastItemLine - firstItemLine + 1
	if oldCount == len(newItemLines) {
		same := true
		for i := 0; i < oldCount; i++ {
			if lines[firstItemLine+i] != newItemLines[i] {
				same = false
				break
			}
		}
		if same {
			return lines, 0, false
		}
	}

	result := make([]string, 0, len(lines)-oldCount+len(newItemLines))
	result = append(result, lines[:firstItemLine]...)
	result = append(result, newItemLines...)
	result = append(result, lines[lastItemLine+1:]...)
	return result, len(newItemLines) - oldCount, true
}

// buildBlockSeq builds lines for a new YAML block sequence field (key + items).
func buildBlockSeq(key string, values []string, keyIndent, itemIndent int) []string {
	result := []string{strings.Repeat(" ", keyIndent) + key + ":"}
	for _, v := range values {
		result = append(result, strings.Repeat(" ", itemIndent)+"- "+marshalYAMLValue(v))
	}
	return result
}

// buildNewBlock marshals a struct value and returns indented lines for "key:\n  field: val\n ...".
func buildNewBlock(key string, value interface{}, keyIndent, indentStep int) []string {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(indentStep)
	_ = enc.Encode(value)
	_ = enc.Close()
	raw := strings.TrimRight(buf.String(), "\n")
	rawLines := strings.Split(raw, "\n")

	fieldPfx := strings.Repeat(" ", keyIndent+indentStep)
	result := []string{strings.Repeat(" ", keyIndent) + key + ":"}
	for _, l := range rawLines {
		if l != "" {
			result = append(result, fieldPfx+l)
		}
	}
	return result
}

// buildNewSingleGroup marshals a Group struct and returns indented lines for one group entry.
func buildNewSingleGroup(name string, g Group, nameIndent, fieldIndent, indentStep int) []string {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(indentStep)
	_ = enc.Encode(&g)
	_ = enc.Close()
	raw := strings.TrimRight(buf.String(), "\n")
	rawLines := strings.Split(raw, "\n")

	fieldPfx := strings.Repeat(" ", fieldIndent)
	result := []string{strings.Repeat(" ", nameIndent) + name + ":"}
	for _, l := range rawLines {
		if l != "" {
			result = append(result, fieldPfx+l)
		}
	}
	return result
}

// buildNewGroupsBlock builds lines for an entire new "groups:" block with all entries.
func buildNewGroupsBlock(groups map[string]Group, keyIndent, indentStep int) []string {
	nameIndent := keyIndent + indentStep
	fieldIndent := nameIndent + indentStep

	result := []string{strings.Repeat(" ", keyIndent) + "groups:"}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		result = append(result, buildNewSingleGroup(name, groups[name], nameIndent, fieldIndent, indentStep)...)
	}
	return result
}

// stringsFromSeqNode extracts a []string from a YAML sequence node.
// Works for both block style (- a\n- b) and flow style ([a, b]).
func stringsFromSeqNode(node *yaml.Node) []string {
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil
	}
	result := make([]string, 0, len(node.Content))
	for _, item := range node.Content {
		if item.Kind == yaml.ScalarNode {
			result = append(result, item.Value)
		}
	}
	return result
}

// groupFromYAMLNode extracts a Group struct from a YAML mapping node.
func groupFromYAMLNode(node *yaml.Node) Group {
	if node == nil || node.Kind != yaml.MappingNode {
		return Group{}
	}
	var g Group
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1]
		switch key {
		case "applies-to":
			g.AppliesTo = val.Value
		case "patterns":
			g.Patterns = stringsFromSeqNode(val)
		case "exclude-patterns":
			g.ExcludePatterns = stringsFromSeqNode(val)
		case "dependency-type":
			g.DependencyType = val.Value
		case "update-types":
			g.UpdateTypes = stringsFromSeqNode(val)
		case "group-by":
			g.GroupBy = val.Value
		}
	}
	return g
}

// stringSlicesEqualSet returns true if two string slices contain the same
// elements regardless of order.
func stringSlicesEqualSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}
	return true
}

// groupsEquivalent returns true if the candidate group's non-empty fields
// all match the corresponding fields in the existing group.
// Slice fields are compared as unordered sets.
func groupsEquivalent(existing, candidate Group) bool {
	if candidate.AppliesTo != "" && existing.AppliesTo != candidate.AppliesTo {
		return false
	}
	if candidate.DependencyType != "" && existing.DependencyType != candidate.DependencyType {
		return false
	}
	if candidate.GroupBy != "" && existing.GroupBy != candidate.GroupBy {
		return false
	}
	if len(candidate.Patterns) > 0 && !stringSlicesEqualSet(existing.Patterns, candidate.Patterns) {
		return false
	}
	if len(candidate.ExcludePatterns) > 0 && !stringSlicesEqualSet(existing.ExcludePatterns, candidate.ExcludePatterns) {
		return false
	}
	if len(candidate.UpdateTypes) > 0 && !stringSlicesEqualSet(existing.UpdateTypes, candidate.UpdateTypes) {
		return false
	}
	return true
}

// buildObjectListBlock marshals a slice of structs and returns indented YAML lines
// for "key:\n  - field: val\n  - field: val\n ...".
func buildObjectListBlock(key string, items interface{}, keyIndent, indentStep int) []string {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(indentStep)
	_ = enc.Encode(items)
	_ = enc.Close()
	raw := strings.TrimRight(buf.String(), "\n")
	rawLines := strings.Split(raw, "\n")

	pfx := strings.Repeat(" ", keyIndent+indentStep)
	result := []string{strings.Repeat(" ", keyIndent) + key + ":"}
	for _, l := range rawLines {
		if l != "" {
			result = append(result, pfx+l)
		}
	}
	return result
}
