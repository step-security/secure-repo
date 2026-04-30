package dependabot

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	dependabot "github.com/paulvollmer/dependabot-config-go"
	"gopkg.in/yaml.v3"
)

type UpdateDependabotConfigResponse struct {
	OriginalInput        string
	FinalOutput          string
	IsChanged            bool
	ConfigfileFetchError bool
}

type Ecosystem struct {
	PackageEcosystem string
	Directory        string
	Directories      []string         `json:",omitempty"`
	Interval         string
	CoolDown         *CoolDown        `json:",omitempty"`
	Groups           map[string]Group `json:",omitempty"`
}

type UpdateDependabotConfigRequest struct {
	Ecosystems  []Ecosystem
	Content     string
	Subtractive bool
}

// CoolDown represents the cooldown block, which the upstream dependabot package does not support.
type CoolDown struct {
	DefaultDays     int      `yaml:"default-days,omitempty"`
	SemverMajorDays int      `yaml:"semver-major-days,omitempty"`
	SemverMinorDays int      `yaml:"semver-minor-days,omitempty"`
	SemverPatchDays int      `yaml:"semver-patch-days,omitempty"`
	Include         []string `yaml:"include,omitempty"`
	Exclude         []string `yaml:"exclude,omitempty"`
}

// Group represents a single entry in the groups block.
type Group struct {
	AppliesTo       string   `yaml:"applies-to,omitempty"`
	Patterns        []string `yaml:"patterns,omitempty"`
	ExcludePatterns []string `yaml:"exclude-patterns,omitempty"`
	DependencyType  string   `yaml:"dependency-type,omitempty"`
	UpdateTypes     []string `yaml:"update-types,omitempty"`
	GroupBy         string   `yaml:"group-by,omitempty"`
}

// ExtendedUpdate embeds the upstream dependabot.Update inline so all its fields are preserved,
// and extends it with the directories, groups, and cooldown blocks.
type ExtendedUpdate struct {
	dependabot.Update `yaml:",inline"`
	Directories       []string         `yaml:"directories,omitempty"`
	Groups            map[string]Group `yaml:"groups,omitempty"`
	CoolDown          *CoolDown        `yaml:"cooldown,omitempty"`
}

// Config is the top-level dependabot config file structure backed by Update.
type Config struct {
	Version int              `yaml:"version"`
	Updates []ExtendedUpdate `yaml:"updates"`
}

// matchesEcosystem reports whether an existing config entry already covers the
// requested ecosystem by package-ecosystem and directory (singular or plural).
func matchesEcosystem(update ExtendedUpdate, eco Ecosystem) bool {
	if update.PackageEcosystem != eco.PackageEcosystem {
		return false
	}
	// Match by singular directory.
	if update.Directory != "" && (update.Directory == eco.Directory || update.Directory == eco.Directory+"/") {
		return true
	}
	// Match by plural directories: covered if any existing directory equals eco.Directory.
	for _, d := range update.Directories {
		if d == eco.Directory || d == eco.Directory+"/" {
			return true
		}
	}
	return false
}

// getIndentation returns the indentation level of the first list found in a given YAML string.
// If the YAML string is empty or invalid, or if no list is found, it returns an error.
func getIndentation(dependabotConfig string) (int, error) {
	// Initialize an empty YAML node
	t := yaml.Node{}

	// Unmarshal the YAML string into the node
	err := yaml.Unmarshal([]byte(dependabotConfig), &t)
	if err != nil {
		return 0, fmt.Errorf("unable to parse yaml: %w", err)
	}

	// Retrieve the top node of the YAML document
	topNode := t.Content
	if len(topNode) == 0 {
		return 0, errors.New("file provided is empty or invalid")
	}

	// Check for the first list and its indentation level
	for _, n := range topNode[0].Content {
		if n.Value == "" && n.Tag == "!!seq" {
			// Return the column of the first list found
			return n.Column, nil
		}
	}

	// Return an error if no list was found
	return 0, errors.New("no list found in yaml")
}

// UpdateDependabotConfig is used to update dependabot configuration and returns an UpdateDependabotConfigResponse.
func UpdateDependabotConfig(dependabotConfig string) (*UpdateDependabotConfigResponse, error) {
	var updateDependabotConfigRequest UpdateDependabotConfigRequest

	// Handle error in json unmarshalling
	err := json.Unmarshal([]byte(dependabotConfig), &updateDependabotConfigRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON from dependabotConfig: %v", err)
	}

	inputConfigFile := []byte(updateDependabotConfigRequest.Content)
	var cfg Config
	err = yaml.Unmarshal(inputConfigFile, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal dependabot config: %v", err)
	}

	indentation := 3

	response := new(UpdateDependabotConfigResponse)
	response.FinalOutput = updateDependabotConfigRequest.Content
	response.OriginalInput = updateDependabotConfigRequest.Content
	response.IsChanged = false

	// In subtractive mode, update only the specified fields of existing entries.
	if updateDependabotConfigRequest.Subtractive {
		if updateDependabotConfigRequest.Content == "" {
			return response, nil
		}
		subtractiveIndent, err := getIndentation(string(inputConfigFile))
		if err != nil {
			return nil, fmt.Errorf("failed to get indentation: %v", err)
		}
		newContent, changed, err := updateSubtractiveFields(response.FinalOutput, updateDependabotConfigRequest.Ecosystems, cfg, subtractiveIndent-1)
		if err != nil {
			return nil, fmt.Errorf("failed to apply subtractive update: %v", err)
		}
		response.FinalOutput = newContent
		response.IsChanged = changed
		return response, nil
	}

	if updateDependabotConfigRequest.Content == "" {
		// Empty content: build from scratch using string concatenation.
		if len(updateDependabotConfigRequest.Ecosystems) == 0 {
			return response, nil
		}
		var finalOutput strings.Builder
		finalOutput.WriteString("version: 2\nupdates:")
		for _, Update := range updateDependabotConfigRequest.Ecosystems {
			item := ExtendedUpdate{
				Update: dependabot.Update{
					PackageEcosystem: Update.PackageEcosystem,
					Directory:        Update.Directory,
					Schedule:         dependabot.Schedule{Interval: Update.Interval},
				},
				Groups:   Update.Groups,
				CoolDown: Update.CoolDown,
			}
			items := []ExtendedUpdate{item}
			addedItem, err := yaml.Marshal(items)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal update items: %v", err)
			}
			data, err := addIndentation(string(addedItem), indentation)
			if err != nil {
				return nil, fmt.Errorf("failed to add indentation: %v", err)
			}
			finalOutput.WriteString(data)
			response.IsChanged = true
		}
		response.FinalOutput = finalOutput.String()
	} else {
		// Non-empty content: insert new entries at the end of the updates section
		// so that sibling top-level keys like registries are preserved in place.
		indentation, err = getIndentation(string(inputConfigFile))
		if err != nil {
			return nil, fmt.Errorf("failed to get indentation: %v", err)
		}

		var rootNode yaml.Node
		if err := yaml.Unmarshal(inputConfigFile, &rootNode); err != nil {
			return nil, fmt.Errorf("failed to parse yaml for insertion point: %v", err)
		}
		if len(rootNode.Content) == 0 {
			return nil, fmt.Errorf("failed to parse yaml: document is empty")
		}
		docNode := rootNode.Content[0]
		updatesNode := findMappingValue(docNode, "updates")
		if updatesNode == nil || updatesNode.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("missing or invalid 'updates' section in dependabot config")
		}

		inputLines := strings.Split(response.FinalOutput, "\n")
		updatesLastLine := findLastLine(updatesNode)
		lineOffset := 0

		// First pass: collect directories to append per existing config entry.
		// This avoids calling replaceSequence multiple times on the same YAML
		// node, which would corrupt line positions.
		dirsToAppend := map[int][]string{} // cfg.Updates index -> dirs to add
		var newEntries []Ecosystem
		for _, Update := range updateDependabotConfigRequest.Ecosystems {
			updateAlreadyExist := false
			for _, update := range cfg.Updates {
				if matchesEcosystem(update, Update) {
					updateAlreadyExist = true
					break
				}
			}

			if !updateAlreadyExist {
				appended := false
				for i, update := range cfg.Updates {
					if update.PackageEcosystem == Update.PackageEcosystem && len(update.Directories) > 0 {
						dirsToAppend[i] = append(dirsToAppend[i], Update.Directory)
						appended = true
						break
					}
				}
				if !appended {
					newEntries = append(newEntries, Update)
				}
			}
		}

		// Apply collected directory appends — one replaceSequence call per entry.
		// Sort indices so we process top-to-bottom; lineOffset stays correct.
		sortedIndices := make([]int, 0, len(dirsToAppend))
		for i := range dirsToAppend {
			sortedIndices = append(sortedIndices, i)
		}
		sort.Ints(sortedIndices)
		for _, i := range sortedIndices {
			dirs := dirsToAppend[i]
			entryNode := updatesNode.Content[i]
			dirsNode := findMappingValue(entryNode, "directories")
			if dirsNode != nil {
				newDirs := uniqueStrings(append(cfg.Updates[i].Directories, dirs...))
				newLines, netChange, ch := replaceSequence(inputLines, dirsNode, newDirs, lineOffset)
				if ch {
					inputLines = newLines
					lineOffset += netChange
					response.IsChanged = true
				}
			}
		}

		for _, Update := range newEntries {
			item := ExtendedUpdate{
				Update: dependabot.Update{
					PackageEcosystem: Update.PackageEcosystem,
					Directory:        Update.Directory,
					Schedule:         dependabot.Schedule{Interval: Update.Interval},
				},
				Directories: Update.Directories,
				Groups:      Update.Groups,
				CoolDown:    Update.CoolDown,
			}
			items := []ExtendedUpdate{item}
			addedItem, err := yaml.Marshal(items)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal update items: %v", err)
			}
			data, err := addIndentation(string(addedItem), indentation)
			if err != nil {
				return nil, fmt.Errorf("failed to add indentation: %v", err)
			}

			// Trim trailing newline to avoid double blank lines when content
			// follows after the updates section (e.g. registries block).
			dataLines := strings.Split(strings.TrimRight(data, "\n"), "\n")
			insertAt := updatesLastLine + lineOffset
			inputLines = insertAfterLine(inputLines, insertAt, dataLines)
			lineOffset += len(dataLines)
			response.IsChanged = true
		}

		response.FinalOutput = strings.Join(inputLines, "\n")
		if !strings.HasSuffix(response.FinalOutput, "\n") {
			response.FinalOutput += "\n"
		}
	}

	return response, nil
}

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
// adding quotes when needed (e.g. '*' becomes '\'*\”).
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

// entryReplacement holds what needs to change for a single updates entry.
type entryReplacement struct {
	entryNode    *yaml.Node
	newInterval  string
	cooldown     *CoolDown
	groups       map[string]Group
	dirsToAppend []string // directories to append to the existing directories list
	existingDirs []string // current directories list (used when dirsToAppend is set)
}

// applyCooldownUpdates updates existing cooldown fields or inserts new ones.
// Processes fields in file order so lineOffset stays accurate.
// Returns (lines inserted, whether changed).
func applyCooldownUpdates(cooldownNode *yaml.Node, cd *CoolDown, lines *[]string, lineOffset, indentStep int) (int, bool) {
	changed := false
	insertedTotal := 0
	fieldIndent := getChildIndent(cooldownNode)

	intUpdates := map[string]int{
		"default-days":      cd.DefaultDays,
		"semver-major-days": cd.SemverMajorDays,
		"semver-minor-days": cd.SemverMinorDays,
		"semver-patch-days": cd.SemverPatchDays,
	}
	seqUpdates := map[string][]string{
		"include": cd.Include,
		"exclude": cd.Exclude,
	}

	// Process existing fields in file order (iterate Content pairs top-to-bottom).
	processed := make(map[string]bool)
	for i := 0; i+1 < len(cooldownNode.Content); i += 2 {
		keyNode := cooldownNode.Content[i]
		valNode := cooldownNode.Content[i+1]
		key := keyNode.Value

		if val, ok := intUpdates[key]; ok && val != 0 {
			processed[key] = true
			valStr := strconv.Itoa(val)
			if valNode.Value != valStr {
				lineIdx := valNode.Line - 1 + lineOffset + insertedTotal
				replaceScalarOnLine(*lines, lineIdx, valNode, valStr)
				changed = true
			}
		}

		if vals, ok := seqUpdates[key]; ok && len(vals) > 0 {
			processed[key] = true
			newLines, netChange, ch := replaceSequence(*lines, valNode, vals, lineOffset+insertedTotal)
			if ch {
				*lines = newLines
				insertedTotal += netChange
				changed = true
			}
		}
	}

	// Add fields that don't exist yet (appended at end of cooldown block).
	for _, key := range []string{"default-days", "semver-major-days", "semver-minor-days", "semver-patch-days"} {
		if processed[key] {
			continue
		}
		val := intUpdates[key]
		if val == 0 {
			continue
		}
		lastLine := findLastLine(cooldownNode) + lineOffset + insertedTotal
		newLine := strings.Repeat(" ", fieldIndent) + key + ": " + strconv.Itoa(val)
		*lines = insertAfterLine(*lines, lastLine, []string{newLine})
		insertedTotal++
		changed = true
	}
	for _, key := range []string{"include", "exclude"} {
		if processed[key] {
			continue
		}
		vals := seqUpdates[key]
		if len(vals) == 0 {
			continue
		}
		lastLine := findLastLine(cooldownNode) + lineOffset + insertedTotal
		newLines := buildBlockSeq(key, vals, fieldIndent, fieldIndent+indentStep)
		*lines = insertAfterLine(*lines, lastLine, newLines)
		insertedTotal += len(newLines)
		changed = true
	}

	return insertedTotal, changed
}

// applyGroupUpdates updates existing group fields or inserts new groups.
// Processes existing groups in file order so lineOffset stays accurate.
// Returns (lines inserted, whether changed).
func applyGroupUpdates(groupsNode *yaml.Node, groups map[string]Group, lines *[]string, lineOffset, indentStep int) (int, bool) {
	changed := false
	insertedTotal := 0
	nameIndent := getChildIndent(groupsNode)
	fieldIndent := nameIndent + indentStep

	// Process existing groups in file order.
	processedGroups := make(map[string]bool)
	for i := 0; i+1 < len(groupsNode.Content); i += 2 {
		groupKeyNode := groupsNode.Content[i]
		groupValNode := groupsNode.Content[i+1]
		groupName := groupKeyNode.Value

		ecoGroup, exists := groups[groupName]
		if !exists {
			continue
		}
		processedGroups[groupName] = true

		grpFieldIndent := getChildIndent(groupValNode)
		if grpFieldIndent == 0 {
			grpFieldIndent = fieldIndent
		}

		// Process fields within this group in file order.
		processedFields := make(map[string]bool)
		for j := 0; j+1 < len(groupValNode.Content); j += 2 {
			fKeyNode := groupValNode.Content[j]
			fValNode := groupValNode.Content[j+1]
			fName := fKeyNode.Value

			switch fName {
			case "applies-to":
				processedFields[fName] = true
				if ecoGroup.AppliesTo != "" && fValNode.Value != ecoGroup.AppliesTo {
					lineIdx := fValNode.Line - 1 + lineOffset + insertedTotal
					replaceScalarOnLine(*lines, lineIdx, fValNode, ecoGroup.AppliesTo)
					changed = true
				}
			case "patterns":
				processedFields[fName] = true
				if len(ecoGroup.Patterns) > 0 {
					nl, nc, ch := replaceSequence(*lines, fValNode, ecoGroup.Patterns, lineOffset+insertedTotal)
					if ch {
						*lines = nl
						insertedTotal += nc
						changed = true
					}
				}
			case "exclude-patterns":
				processedFields[fName] = true
				if len(ecoGroup.ExcludePatterns) > 0 {
					nl, nc, ch := replaceSequence(*lines, fValNode, ecoGroup.ExcludePatterns, lineOffset+insertedTotal)
					if ch {
						*lines = nl
						insertedTotal += nc
						changed = true
					}
				}
			case "dependency-type":
				processedFields[fName] = true
				if ecoGroup.DependencyType != "" && fValNode.Value != ecoGroup.DependencyType {
					lineIdx := fValNode.Line - 1 + lineOffset + insertedTotal
					replaceScalarOnLine(*lines, lineIdx, fValNode, ecoGroup.DependencyType)
					changed = true
				}
			case "update-types":
				processedFields[fName] = true
				if len(ecoGroup.UpdateTypes) > 0 {
					nl, nc, ch := replaceSequence(*lines, fValNode, ecoGroup.UpdateTypes, lineOffset+insertedTotal)
					if ch {
						*lines = nl
						insertedTotal += nc
						changed = true
					}
				}
			case "group-by":
				processedFields[fName] = true
				if ecoGroup.GroupBy != "" && fValNode.Value != ecoGroup.GroupBy {
					lineIdx := fValNode.Line - 1 + lineOffset + insertedTotal
					replaceScalarOnLine(*lines, lineIdx, fValNode, ecoGroup.GroupBy)
					changed = true
				}
			}
		}

		// Add fields that don't exist in this group.
		addScalar := func(key, value string) {
			if value == "" || processedFields[key] {
				return
			}
			ll := findLastLine(groupValNode) + lineOffset + insertedTotal
			*lines = insertAfterLine(*lines, ll, []string{strings.Repeat(" ", grpFieldIndent) + key + ": " + value})
			insertedTotal++
			changed = true
		}
		addSeq := func(key string, values []string) {
			if len(values) == 0 || processedFields[key] {
				return
			}
			ll := findLastLine(groupValNode) + lineOffset + insertedTotal
			nl := buildBlockSeq(key, values, grpFieldIndent, grpFieldIndent+indentStep)
			*lines = insertAfterLine(*lines, ll, nl)
			insertedTotal += len(nl)
			changed = true
		}
		addScalar("applies-to", ecoGroup.AppliesTo)
		addSeq("patterns", ecoGroup.Patterns)
		addSeq("exclude-patterns", ecoGroup.ExcludePatterns)
		addScalar("dependency-type", ecoGroup.DependencyType)
		addSeq("update-types", ecoGroup.UpdateTypes)
		addScalar("group-by", ecoGroup.GroupBy)
	}

	// Add new groups (sorted for deterministic output).
	var newGroupNames []string
	for name := range groups {
		if !processedGroups[name] {
			newGroupNames = append(newGroupNames, name)
		}
	}
	sort.Strings(newGroupNames)
	for _, name := range newGroupNames {
		lastLine := findLastLine(groupsNode) + lineOffset + insertedTotal
		newLines := buildNewSingleGroup(name, groups[name], nameIndent, fieldIndent, indentStep)
		*lines = insertAfterLine(*lines, lastLine, newLines)
		insertedTotal += len(newLines)
		changed = true
	}

	return insertedTotal, changed
}

// updateSubtractiveFields finds each ecosystem entry in the existing YAML config by
// PackageEcosystem + Directory, then updates only the non-empty fields from the request.
// It uses the yaml.Node tree for navigation (line numbers) and does targeted text edits
// on the original lines — preserving blank lines, comments, registries, and all formatting.
func updateSubtractiveFields(content string, ecosystems []Ecosystem, cfg Config, indent int) (string, bool, error) {
	var rootNode yaml.Node
	if err := yaml.Unmarshal([]byte(content), &rootNode); err != nil {
		return "", false, fmt.Errorf("failed to parse yaml: %w", err)
	}
	if len(rootNode.Content) == 0 {
		return content, false, nil
	}
	docNode := rootNode.Content[0]

	updatesNode := findMappingValue(docNode, "updates")
	if updatesNode == nil || updatesNode.Kind != yaml.SequenceNode {
		return content, false, nil
	}

	// Phase 1: Build replacements by matching request ecosystems to YAML entries.
	// Uses the Config struct for matching (like the additive path and maintainedActions),
	// and updatesNode.Content[i] to get the corresponding yaml.Node for line-based edits.
	// replacementMap is keyed on *yaml.Node so that multiple ecosystems targeting the same
	// YAML entry (e.g. npm "/" and npm "/temp" both referencing the same npm directories
	// block) are merged into a single entryReplacement, preventing double-edit corruption.
	replacementMap := make(map[*yaml.Node]*entryReplacement)
	var toAdd []Ecosystem

	mergeInto := func(r *entryReplacement, eco Ecosystem) {
		if r.newInterval == "" {
			r.newInterval = eco.Interval
		}
		if r.cooldown == nil {
			r.cooldown = eco.CoolDown
		}
		if r.groups == nil {
			r.groups = eco.Groups
		}
	}

	for _, eco := range ecosystems {
		found := false
		for i, update := range cfg.Updates {
			if matchesEcosystem(update, eco) {
				found = true
				node := updatesNode.Content[i]
				if existing, ok := replacementMap[node]; ok {
					mergeInto(existing, eco)
				} else {
					replacementMap[node] = &entryReplacement{
						entryNode:   node,
						newInterval: eco.Interval,
						cooldown:    eco.CoolDown,
						groups:      eco.Groups,
					}
				}
				break
			}
		}
		if !found {
			// If an existing entry for the same ecosystem uses directories (plural),
			// append the new directory to that list and apply the field updates there,
			// instead of creating a separate new entry.
			appendedTo := false
			for i, update := range cfg.Updates {
				if update.PackageEcosystem == eco.PackageEcosystem && len(update.Directories) > 0 {
					node := updatesNode.Content[i]
					if existing, ok := replacementMap[node]; ok {
						existing.dirsToAppend = append(existing.dirsToAppend, eco.Directory)
						if existing.existingDirs == nil {
							existing.existingDirs = update.Directories
						}
						mergeInto(existing, eco)
					} else {
						replacementMap[node] = &entryReplacement{
							entryNode:    node,
							newInterval:  eco.Interval,
							cooldown:     eco.CoolDown,
							groups:       eco.Groups,
							dirsToAppend: []string{eco.Directory},
							existingDirs: update.Directories,
						}
					}
					appendedTo = true
					break
				}
			}
			if !appendedTo {
				toAdd = append(toAdd, eco)
			}
		}
	}

	var replacements []entryReplacement
	for _, r := range replacementMap {
		replacements = append(replacements, *r)
	}

	if len(replacements) == 0 && len(toAdd) == 0 {
		return content, false, nil
	}

	// Sort replacements by line number so we process top-to-bottom.
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].entryNode.Line < replacements[j].entryNode.Line
	})

	// Phase 2: Apply replacements on original lines.
	// For each entry we first update all EXISTING fields (no new blocks), then
	// insert any NEW blocks at the end. This prevents line-offset confusion when
	// an insertion shifts lines that later operations still reference.
	inputLines := strings.Split(content, "\n")
	changed := false
	lineOffset := 0

	for _, r := range replacements {
		keyIndent := r.entryNode.Column - 1

		// --- Append directories to existing directories list ---
		if len(r.dirsToAppend) > 0 {
			dirsNode := findMappingValue(r.entryNode, "directories")
			if dirsNode != nil {
				newDirs := append(r.existingDirs, r.dirsToAppend...)
				newLines, netChange, ch := replaceSequence(inputLines, dirsNode, newDirs, lineOffset)
				if ch {
					inputLines = newLines
					lineOffset += netChange
					changed = true
				}
			}
		}

		// --- Interval ---
		if r.newInterval != "" {
			schedNode := findMappingValue(r.entryNode, "schedule")
			if schedNode != nil {
				intervalNode := findMappingValue(schedNode, "interval")
				if intervalNode != nil && intervalNode.Value != r.newInterval {
					lineIdx := intervalNode.Line - 1 + lineOffset
					replaceScalarOnLine(inputLines, lineIdx, intervalNode, r.newInterval)
					changed = true
				}
			}
		}

		// --- Update EXISTING cooldown and groups in file order ---
		existingCooldownNode := findMappingValue(r.entryNode, "cooldown")
		existingGroupsNode := findMappingValue(r.entryNode, "groups")

		// Determine which appears first in the file.
		cooldownFirst := true
		if existingCooldownNode != nil && existingGroupsNode != nil {
			cdLine, grpLine := 0, 0
			for i := 0; i+1 < len(r.entryNode.Content); i += 2 {
				if r.entryNode.Content[i].Value == "cooldown" {
					cdLine = r.entryNode.Content[i].Line
				}
				if r.entryNode.Content[i].Value == "groups" {
					grpLine = r.entryNode.Content[i].Line
				}
			}
			cooldownFirst = cdLine < grpLine
		}

		processExistingCooldown := func() {
			if r.cooldown != nil && existingCooldownNode != nil {
				lo, ch := applyCooldownUpdates(existingCooldownNode, r.cooldown, &inputLines, lineOffset, indent)
				lineOffset += lo
				if ch {
					changed = true
				}
			}
		}
		processExistingGroups := func() {
			if len(r.groups) > 0 && existingGroupsNode != nil {
				lo, ch := applyGroupUpdates(existingGroupsNode, r.groups, &inputLines, lineOffset, indent)
				lineOffset += lo
				if ch {
					changed = true
				}
			}
		}

		if cooldownFirst {
			processExistingCooldown()
			processExistingGroups()
		} else {
			processExistingGroups()
			processExistingCooldown()
		}

		// --- Insert NEW blocks at end of entry ---
		if r.cooldown != nil && existingCooldownNode == nil {
			insertAfter := findLastLine(r.entryNode) + lineOffset
			newLines := buildNewBlock("cooldown", r.cooldown, keyIndent, indent)
			inputLines = insertAfterLine(inputLines, insertAfter, newLines)
			lineOffset += len(newLines)
			changed = true
		}
		if len(r.groups) > 0 && existingGroupsNode == nil {
			entryLastLine := findLastLine(r.entryNode) + lineOffset
			newLines := buildNewGroupsBlock(r.groups, keyIndent, indent)
			inputLines = insertAfterLine(inputLines, entryLastLine, newLines)
			lineOffset += len(newLines)
			changed = true
		}
	}

	if !changed && len(toAdd) == 0 {
		return content, false, nil
	}

	// Insert ecosystems that were not found in existing config at the end of the
	// updates section, so sibling top-level keys like registries stay in place.
	updatesLastLine := findLastLine(updatesNode) + lineOffset
	for _, eco := range toAdd {
		item := ExtendedUpdate{
			Update: dependabot.Update{
				PackageEcosystem: eco.PackageEcosystem,
				Directory:        eco.Directory,
				Schedule:         dependabot.Schedule{Interval: eco.Interval},
			},
			Groups:   eco.Groups,
			CoolDown: eco.CoolDown,
		}
		items := []ExtendedUpdate{item}
		var itemBuf bytes.Buffer
		itemEnc := yaml.NewEncoder(&itemBuf)
		itemEnc.SetIndent(indent)
		if err := itemEnc.Encode(items); err != nil {
			return "", false, fmt.Errorf("failed to marshal update items: %w", err)
		}
		data, err := addIndentation(itemBuf.String(), indent+1)
		if err != nil {
			return "", false, fmt.Errorf("failed to add indentation: %w", err)
		}

		dataLines := strings.Split(strings.TrimRight(data, "\n"), "\n")
		inputLines = insertAfterLine(inputLines, updatesLastLine, dataLines)
		updatesLastLine += len(dataLines)
		changed = true
	}

	result := strings.Join(inputLines, "\n")
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result, true, nil
}

// addIndentation adds a certain number of spaces to the start of each line in the input string.
// It returns a new string with the added indentation.
func addIndentation(data string, indentation int) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	var finalData strings.Builder

	// Create the indentation string
	spaces := strings.Repeat(" ", indentation-1)

	finalData.WriteString("\n")

	// Add indentation to each line
	for scanner.Scan() {
		finalData.WriteString(spaces)
		finalData.WriteString(scanner.Text())
		finalData.WriteString("\n")
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error during scanning: %w", err)
	}

	return finalData.String(), nil
}

// uniqueStrings removes duplicate strings from a slice while preserving order.
func uniqueStrings(s []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
