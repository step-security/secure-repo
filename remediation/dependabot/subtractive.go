package dependabot

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	dependabot "github.com/paulvollmer/dependabot-config-go"
	"gopkg.in/yaml.v3"
)

// entryReplacement holds what needs to change for a single updates entry.
type entryReplacement struct {
	entryNode *yaml.Node
	eco       Ecosystem
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

// applyCommitMessageUpdate updates existing commit-message fields or inserts new ones.
// Returns (lines inserted, whether changed).
func applyCommitMessageUpdate(cmNode *yaml.Node, cm *dependabot.CommitMessage, lines *[]string, lineOffset, indentStep int) (int, bool) {
	changed := false
	insertedTotal := 0
	fieldIndent := getChildIndent(cmNode)

	updates := map[string]string{
		"prefix":             cm.Prefix,
		"prefix-development": cm.PrefixDevelopment,
		"include":            cm.Include,
	}

	processed := make(map[string]bool)
	for i := 0; i+1 < len(cmNode.Content); i += 2 {
		keyNode := cmNode.Content[i]
		valNode := cmNode.Content[i+1]
		key := keyNode.Value

		if val, ok := updates[key]; ok && val != "" {
			processed[key] = true
			if valNode.Value != val {
				lineIdx := valNode.Line - 1 + lineOffset + insertedTotal
				replaceScalarOnLine(*lines, lineIdx, valNode, val)
				changed = true
			}
		}
	}

	for _, key := range []string{"prefix", "prefix-development", "include"} {
		if processed[key] || updates[key] == "" {
			continue
		}
		lastLine := findLastLine(cmNode) + lineOffset + insertedTotal
		newLine := strings.Repeat(" ", fieldIndent) + key + ": " + marshalYAMLValue(updates[key])
		*lines = insertAfterLine(*lines, lastLine, []string{newLine})
		insertedTotal++
		changed = true
	}

	return insertedTotal, changed
}

// replaceObjectList replaces an existing YAML sequence of mapping items with new items.
// Returns (net line count change, whether changed).
func replaceObjectList(lines *[]string, seqNode *yaml.Node, items interface{}, lineOffset, indentStep int) (int, bool) {
	if len(seqNode.Content) == 0 {
		return 0, false
	}

	firstLine := seqNode.Content[0].Line - 1 + lineOffset
	lastLine := findLastLine(seqNode) - 1 + lineOffset
	itemIndent := seqNode.Column - 1

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(indentStep)
	_ = enc.Encode(items)
	_ = enc.Close()
	raw := strings.TrimRight(buf.String(), "\n")
	rawLines := strings.Split(raw, "\n")

	pfx := strings.Repeat(" ", itemIndent)
	newLines := make([]string, 0, len(rawLines))
	for _, l := range rawLines {
		if l != "" {
			newLines = append(newLines, pfx+l)
		}
	}

	oldCount := lastLine - firstLine + 1
	if oldCount == len(newLines) {
		same := true
		for i := 0; i < oldCount; i++ {
			if (*lines)[firstLine+i] != newLines[i] {
				same = false
				break
			}
		}
		if same {
			return 0, false
		}
	}

	result := make([]string, 0, len(*lines)-oldCount+len(newLines))
	result = append(result, (*lines)[:firstLine]...)
	result = append(result, newLines...)
	result = append(result, (*lines)[lastLine+1:]...)
	*lines = result
	return len(newLines) - oldCount, true
}

// applyEntryReplacements iterates an entry node's fields in file order and updates
// existing values that differ from the requested Ecosystem. Returns the updated lines,
// lineOffset, changed flag, and populates processed with keys that were found.
func applyEntryReplacements(r entryReplacement, inputLines []string, lineOffset int, changed bool, processed map[string]bool, indent int) ([]string, int, bool) {
	for i := 0; i+1 < len(r.entryNode.Content); i += 2 {
		key := r.entryNode.Content[i].Value
		valNode := r.entryNode.Content[i+1]

		switch key {
		case "schedule":
			processed[key] = true
			if r.eco.Interval != "" {
				intervalNode := findMappingValue(valNode, "interval")
				if intervalNode != nil && intervalNode.Value != r.eco.Interval {
					lineIdx := intervalNode.Line - 1 + lineOffset
					replaceScalarOnLine(inputLines, lineIdx, intervalNode, r.eco.Interval)
					changed = true
				}
			}
			schedSubFields := []struct{ k, v string }{
				{"day", r.eco.Day}, {"time", r.eco.Time}, {"timezone", r.eco.Timezone},
			}
			for _, sf := range schedSubFields {
				if sf.v == "" {
					continue
				}
				sfNode := findMappingValue(valNode, sf.k)
				if sfNode != nil {
					if sfNode.Value != sf.v {
						lineIdx := sfNode.Line - 1 + lineOffset
						replaceScalarOnLine(inputLines, lineIdx, sfNode, sf.v)
						changed = true
					}
				} else {
					fieldIndent := getChildIndent(valNode)
					lastLine := findLastLine(valNode) + lineOffset
					newLine := strings.Repeat(" ", fieldIndent) + sf.k + ": " + marshalYAMLValue(sf.v)
					inputLines = insertAfterLine(inputLines, lastLine, []string{newLine})
					lineOffset++
					changed = true
				}
			}

		case "cooldown":
			processed[key] = true
			if r.eco.CoolDown != nil {
				lo, ch := applyCooldownUpdates(valNode, r.eco.CoolDown, &inputLines, lineOffset, indent)
				lineOffset += lo
				if ch {
					changed = true
				}
			}

		case "groups":
			processed[key] = true
			if len(r.eco.Groups) > 0 {
				lo, ch := applyGroupUpdates(valNode, r.eco.Groups, &inputLines, lineOffset, indent)
				lineOffset += lo
				if ch {
					changed = true
				}
			}

		case "rebase-strategy":
			processed[key] = true
			if r.eco.RebaseStrategy != "" && valNode.Value != r.eco.RebaseStrategy {
				lineIdx := valNode.Line - 1 + lineOffset
				replaceScalarOnLine(inputLines, lineIdx, valNode, r.eco.RebaseStrategy)
				changed = true
			}

		case "target-branch":
			processed[key] = true
			if r.eco.TargetBranch != "" && valNode.Value != r.eco.TargetBranch {
				lineIdx := valNode.Line - 1 + lineOffset
				replaceScalarOnLine(inputLines, lineIdx, valNode, r.eco.TargetBranch)
				changed = true
			}

		case "versioning-strategy":
			processed[key] = true
			if r.eco.VersioningStrategy != "" && valNode.Value != r.eco.VersioningStrategy {
				lineIdx := valNode.Line - 1 + lineOffset
				replaceScalarOnLine(inputLines, lineIdx, valNode, r.eco.VersioningStrategy)
				changed = true
			}

		case "milestone":
			processed[key] = true
			if r.eco.Milestone != nil {
				newVal := strconv.Itoa(*r.eco.Milestone)
				if valNode.Value != newVal {
					lineIdx := valNode.Line - 1 + lineOffset
					replaceScalarOnLine(inputLines, lineIdx, valNode, newVal)
					changed = true
				}
			}

		case "open-pull-requests-limit":
			processed[key] = true
			if r.eco.OpenPullRequestsLimit != nil {
				newVal := strconv.Itoa(*r.eco.OpenPullRequestsLimit)
				if valNode.Value != newVal {
					lineIdx := valNode.Line - 1 + lineOffset
					replaceScalarOnLine(inputLines, lineIdx, valNode, newVal)
					changed = true
				}
			}

		case "assignees":
			processed[key] = true
			if len(r.eco.Assignees) > 0 {
				nl, nc, ch := replaceSequence(inputLines, valNode, r.eco.Assignees, lineOffset)
				if ch {
					inputLines = nl
					lineOffset += nc
					changed = true
				}
			}

		case "reviewers":
			processed[key] = true
			if len(r.eco.Reviewers) > 0 {
				nl, nc, ch := replaceSequence(inputLines, valNode, r.eco.Reviewers, lineOffset)
				if ch {
					inputLines = nl
					lineOffset += nc
					changed = true
				}
			}

		case "labels":
			processed[key] = true
			if len(r.eco.Labels) > 0 {
				nl, nc, ch := replaceSequence(inputLines, valNode, r.eco.Labels, lineOffset)
				if ch {
					inputLines = nl
					lineOffset += nc
					changed = true
				}
			}

		case "commit-message":
			processed[key] = true
			if r.eco.CommitMessage != nil {
				lo, ch := applyCommitMessageUpdate(valNode, r.eco.CommitMessage, &inputLines, lineOffset, indent)
				lineOffset += lo
				if ch {
					changed = true
				}
			}

		case "pull-request-branch-name":
			processed[key] = true
			if r.eco.PullRequestBranchName != nil {
				sepNode := findMappingValue(valNode, "separator")
				if sepNode != nil && sepNode.Value != r.eco.PullRequestBranchName.Separator {
					lineIdx := sepNode.Line - 1 + lineOffset
					replaceScalarOnLine(inputLines, lineIdx, sepNode, r.eco.PullRequestBranchName.Separator)
					changed = true
				}
			}

		case "allow":
			processed[key] = true
			if len(r.eco.Allow) > 0 && valNode.Kind == yaml.SequenceNode {
				lo, ch := replaceObjectList(&inputLines, valNode, r.eco.Allow, lineOffset, indent)
				if ch {
					lineOffset += lo
					changed = true
				}
			}

		case "ignore":
			processed[key] = true
			if len(r.eco.Ignore) > 0 && valNode.Kind == yaml.SequenceNode {
				lo, ch := replaceObjectList(&inputLines, valNode, r.eco.Ignore, lineOffset, indent)
				if ch {
					lineOffset += lo
					changed = true
				}
			}

		case "registries":
			processed[key] = true
			if len(r.eco.Registries) > 0 {
				nl, nc, ch := replaceSequence(inputLines, valNode, r.eco.Registries, lineOffset)
				if ch {
					inputLines = nl
					lineOffset += nc
					changed = true
				}
			}

		case "exclude-paths":
			processed[key] = true
			if len(r.eco.ExcludePaths) > 0 {
				nl, nc, ch := replaceSequence(inputLines, valNode, r.eco.ExcludePaths, lineOffset)
				if ch {
					inputLines = nl
					lineOffset += nc
					changed = true
				}
			}

		case "directories":
			processed[key] = true
			if len(r.eco.Directories) > 0 {
				nl, nc, ch := replaceSequence(inputLines, valNode, r.eco.Directories, lineOffset)
				if ch {
					inputLines = nl
					lineOffset += nc
					changed = true
				}
			}

		case "vendor":
			processed[key] = true
			if r.eco.Vendor != nil {
				newVal := strconv.FormatBool(*r.eco.Vendor)
				if valNode.Value != newVal {
					lineIdx := valNode.Line - 1 + lineOffset
					replaceScalarOnLine(inputLines, lineIdx, valNode, newVal)
					changed = true
				}
			}

		case "insecure-external-code-execution":
			processed[key] = true
			if r.eco.InsecureExternalCodeExecution != "" && valNode.Value != r.eco.InsecureExternalCodeExecution {
				lineIdx := valNode.Line - 1 + lineOffset
				replaceScalarOnLine(inputLines, lineIdx, valNode, r.eco.InsecureExternalCodeExecution)
				changed = true
			}

		case "multi-ecosystem-group":
			processed[key] = true
			if r.eco.MultiEcosystemGroup != "" && valNode.Value != r.eco.MultiEcosystemGroup {
				lineIdx := valNode.Line - 1 + lineOffset
				replaceScalarOnLine(inputLines, lineIdx, valNode, r.eco.MultiEcosystemGroup)
				changed = true
			}

		case "enable-beta-ecosystems":
			processed[key] = true
			if r.eco.EnableBetaEcosystems != nil {
				newVal := strconv.FormatBool(*r.eco.EnableBetaEcosystems)
				if valNode.Value != newVal {
					lineIdx := valNode.Line - 1 + lineOffset
					replaceScalarOnLine(inputLines, lineIdx, valNode, newVal)
					changed = true
				}
			}
		}
	}

	return inputLines, lineOffset, changed
}

// insertNewEntryFields appends fields from the Ecosystem that were not found in the
// existing YAML entry (i.e. not in processed). Returns updated lines, lineOffset, changed.
func insertNewEntryFields(r entryReplacement, inputLines []string, lineOffset int, changed bool, processed map[string]bool, indent int) ([]string, int, bool) {
	keyIndent := r.entryNode.Column - 1
	entryLastLine := findLastLine(r.entryNode) + lineOffset

	insertBlock := func(key string, value interface{}) {
		newLines := buildNewBlock(key, value, keyIndent, indent)
		inputLines = insertAfterLine(inputLines, entryLastLine, newLines)
		lineOffset += len(newLines)
		entryLastLine += len(newLines)
		changed = true
	}
	insertScalar := func(key, value string) {
		newLine := strings.Repeat(" ", keyIndent) + key + ": " + marshalYAMLValue(value)
		inputLines = insertAfterLine(inputLines, entryLastLine, []string{newLine})
		lineOffset++
		entryLastLine++
		changed = true
	}
	insertIntScalar := func(key string, value int) {
		newLine := strings.Repeat(" ", keyIndent) + key + ": " + strconv.Itoa(value)
		inputLines = insertAfterLine(inputLines, entryLastLine, []string{newLine})
		lineOffset++
		entryLastLine++
		changed = true
	}
	insertStringList := func(key string, values []string) {
		newLines := buildBlockSeq(key, values, keyIndent, keyIndent+indent)
		inputLines = insertAfterLine(inputLines, entryLastLine, newLines)
		lineOffset += len(newLines)
		entryLastLine += len(newLines)
		changed = true
	}
	insertObjectList := func(key string, items interface{}) {
		newLines := buildObjectListBlock(key, items, keyIndent, indent)
		inputLines = insertAfterLine(inputLines, entryLastLine, newLines)
		lineOffset += len(newLines)
		entryLastLine += len(newLines)
		changed = true
	}
	insertBoolScalar := func(key string, value bool) {
		newLine := strings.Repeat(" ", keyIndent) + key + ": " + strconv.FormatBool(value)
		inputLines = insertAfterLine(inputLines, entryLastLine, []string{newLine})
		lineOffset++
		entryLastLine++
		changed = true
	}

	if r.eco.CoolDown != nil && !processed["cooldown"] {
		insertBlock("cooldown", r.eco.CoolDown)
	}
	if len(r.eco.Groups) > 0 && !processed["groups"] {
		newLines := buildNewGroupsBlock(r.eco.Groups, keyIndent, indent)
		inputLines = insertAfterLine(inputLines, entryLastLine, newLines)
		lineOffset += len(newLines)
		entryLastLine += len(newLines)
		changed = true
	}
	if r.eco.RebaseStrategy != "" && !processed["rebase-strategy"] {
		insertScalar("rebase-strategy", r.eco.RebaseStrategy)
	}
	if r.eco.TargetBranch != "" && !processed["target-branch"] {
		insertScalar("target-branch", r.eco.TargetBranch)
	}
	if r.eco.VersioningStrategy != "" && !processed["versioning-strategy"] {
		insertScalar("versioning-strategy", r.eco.VersioningStrategy)
	}
	if r.eco.Milestone != nil && !processed["milestone"] {
		insertIntScalar("milestone", *r.eco.Milestone)
	}
	if r.eco.OpenPullRequestsLimit != nil && !processed["open-pull-requests-limit"] {
		insertIntScalar("open-pull-requests-limit", *r.eco.OpenPullRequestsLimit)
	}
	if len(r.eco.Assignees) > 0 && !processed["assignees"] {
		insertStringList("assignees", r.eco.Assignees)
	}
	if len(r.eco.Reviewers) > 0 && !processed["reviewers"] {
		insertStringList("reviewers", r.eco.Reviewers)
	}
	if len(r.eco.Labels) > 0 && !processed["labels"] {
		insertStringList("labels", r.eco.Labels)
	}
	if r.eco.CommitMessage != nil && !processed["commit-message"] {
		insertBlock("commit-message", r.eco.CommitMessage)
	}
	if r.eco.PullRequestBranchName != nil && !processed["pull-request-branch-name"] {
		insertBlock("pull-request-branch-name", r.eco.PullRequestBranchName)
	}
	if len(r.eco.Allow) > 0 && !processed["allow"] {
		insertObjectList("allow", r.eco.Allow)
	}
	if len(r.eco.Ignore) > 0 && !processed["ignore"] {
		insertObjectList("ignore", r.eco.Ignore)
	}
	if len(r.eco.Registries) > 0 && !processed["registries"] {
		insertStringList("registries", r.eco.Registries)
	}
	if len(r.eco.ExcludePaths) > 0 && !processed["exclude-paths"] {
		insertStringList("exclude-paths", r.eco.ExcludePaths)
	}
	if len(r.eco.Directories) > 0 && !processed["directories"] {
		insertStringList("directories", r.eco.Directories)
	}
	if r.eco.Vendor != nil && !processed["vendor"] {
		insertBoolScalar("vendor", *r.eco.Vendor)
	}
	if r.eco.InsecureExternalCodeExecution != "" && !processed["insecure-external-code-execution"] {
		insertScalar("insecure-external-code-execution", r.eco.InsecureExternalCodeExecution)
	}
	if r.eco.MultiEcosystemGroup != "" && !processed["multi-ecosystem-group"] {
		insertScalar("multi-ecosystem-group", r.eco.MultiEcosystemGroup)
	}
	if r.eco.EnableBetaEcosystems != nil && !processed["enable-beta-ecosystems"] {
		insertBoolScalar("enable-beta-ecosystems", *r.eco.EnableBetaEcosystems)
	}

	return inputLines, lineOffset, changed
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
	// replacementByNode tracks which YAML node already has a replacement so that
	// multiple ecosystems targeting the same entry (e.g. npm "/" and npm "/temp"
	// both referencing the same npm directories block) are merged, preventing
	// double-edit corruption.
	var replacements []entryReplacement
	replacementByNode := make(map[*yaml.Node]int) // node → index in replacements
	var toAdd []Ecosystem

	for _, eco := range ecosystems {
		found := false
		for i, update := range cfg.Updates {
			if matchesEcosystem(update, eco) {
				found = true
				node := updatesNode.Content[i]
				if idx, ok := replacementByNode[node]; ok {
					// Merge: append this eco's directory to existing replacement's Directories.
					existing := &replacements[idx]
					if eco.Directory != "" {
						existing.eco.Directories = append(existing.eco.Directories, eco.Directory)
					}
				} else {
					replacementByNode[node] = len(replacements)
					replacements = append(replacements, entryReplacement{
						entryNode: node,
						eco:       eco,
					})
				}
				break
			}
		}
		if !found {
			// If an existing entry for the same ecosystem uses directories (plural),
			// append the new directory to that list instead of creating a separate new entry.
			appendedTo := false
			for i, update := range cfg.Updates {
				if update.PackageEcosystem == eco.PackageEcosystem && len(update.Directories) > 0 {
					node := updatesNode.Content[i]
					if idx, ok := replacementByNode[node]; ok {
						existing := &replacements[idx]
						if eco.Directory != "" {
							existing.eco.Directories = append(existing.eco.Directories, eco.Directory)
						}
					} else {
						// Create replacement with Directories = existing + new.
						mergedEco := eco
						mergedEco.Directories = make([]string, 0, len(update.Directories)+1)
						mergedEco.Directories = append(mergedEco.Directories, update.Directories...)
						if eco.Directory != "" {
							mergedEco.Directories = append(mergedEco.Directories, eco.Directory)
						}
						replacementByNode[node] = len(replacements)
						replacements = append(replacements, entryReplacement{
							entryNode: node,
							eco:       mergedEco,
						})
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

	if len(replacements) == 0 && len(toAdd) == 0 {
		return content, false, nil
	}

	// Sort replacements by line number so we process top-to-bottom.
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].entryNode.Line < replacements[j].entryNode.Line
	})

	// Phase 2: Apply replacements on original lines.
	// Iterate each entry node's Content in file order so lineOffset stays accurate
	// regardless of how many fields each entry has or what order they appear.
	inputLines := strings.Split(content, "\n")
	changed := false
	lineOffset := 0

	for _, r := range replacements {
		processed := make(map[string]bool)

		inputLines, lineOffset, changed = applyEntryReplacements(r, inputLines, lineOffset, changed, processed, indent)
		inputLines, lineOffset, changed = insertNewEntryFields(r, inputLines, lineOffset, changed, processed, indent)
	}

	if !changed && len(toAdd) == 0 {
		return content, false, nil
	}

	// Insert ecosystems that were not found in existing config at the end of the
	// updates section, so sibling top-level keys like registries stay in place.
	updatesLastLine := findLastLine(updatesNode) + lineOffset
	for _, eco := range toAdd {
		item := ecosystemToExtendedUpdate(eco)
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
