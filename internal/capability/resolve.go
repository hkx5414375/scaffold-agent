// Package capability resolves deterministic capability dependency graphs.
package capability

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

// Catalog stores every supported version of a capability by name and semantic version.
type Catalog map[string]map[string]spec.CapabilityPack

// NewCatalog builds an immutable-by-convention versioned catalog from capability documents.
func NewCatalog(packs ...spec.CapabilityPack) Catalog {
	catalog := make(Catalog)
	for _, pack := range packs {
		versions := catalog[pack.Metadata.Name]
		if versions == nil {
			versions = make(map[string]spec.CapabilityPack)
			catalog[pack.Metadata.Name] = versions
		}
		versions[pack.Metadata.Version] = pack
	}
	return catalog
}

type requirement struct {
	constraint string
	source     string
}

type resolutionIssue struct {
	code    string
	path    string
	message string
}

// Resolve returns exact selected packs and deterministic transitive dependencies in dependency-first order.
// Dependency ranges choose the highest catalog version that satisfies the complete constraint intersection.
func Resolve(catalog Catalog, selected []spec.CapabilitySelection) ([]spec.CapabilityPack, []spec.Diagnostic) {
	var diagnostics []spec.Diagnostic
	requirements := make(map[string][]requirement, len(selected))
	requested := make(map[string]struct{}, len(selected))
	for _, selection := range selected {
		if _, exists := requested[selection.Name]; exists {
			diagnostics = append(diagnostics, diagnostic("capability.selection.duplicate", selection.Name, fmt.Sprintf("capability %q is selected more than once", selection.Name)))
			continue
		}
		requested[selection.Name] = struct{}{}
		if selection.Version == "" {
			diagnostics = append(diagnostics, diagnostic("capability.version.required", selection.Name, fmt.Sprintf("capability %q must pin an exact version", selection.Name)))
			continue
		}
		requirements[selection.Name] = append(requirements[selection.Name], requirement{
			constraint: selection.Version,
			source:     "project selection",
		})
	}
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}

	roots := make([]string, 0, len(requested))
	for name := range requested {
		roots = append(roots, name)
	}
	sort.Strings(roots)
	resolved, issue := search(catalog, requirements, make(map[string]spec.CapabilityPack))
	if issue != nil {
		return nil, []spec.Diagnostic{diagnostic(issue.code, issue.path, issue.message)}
	}

	state := make(map[string]uint8, len(resolved))
	var order []spec.CapabilityPack
	var visit func(string)
	visit = func(name string) {
		switch state[name] {
		case 1:
			diagnostics = append(diagnostics, diagnostic("capability.dependency.cycle", name, fmt.Sprintf("capability dependency cycle includes %q", name)))
			return
		case 2:
			return
		}
		state[name] = 1
		pack := resolved[name]
		dependencies := append([]spec.PackDependency(nil), pack.Spec.Requires...)
		sort.Slice(dependencies, func(i, j int) bool { return dependencies[i].Name < dependencies[j].Name })
		for _, dependency := range dependencies {
			visit(dependency.Name)
		}
		state[name] = 2
		order = append(order, pack)
	}
	for _, name := range roots {
		visit(name)
	}

	closure := make(map[string]struct{}, len(order))
	for _, pack := range order {
		closure[pack.Metadata.Name] = struct{}{}
	}
	for _, pack := range order {
		conflicts := append([]string(nil), pack.Spec.Conflicts...)
		sort.Strings(conflicts)
		for _, conflict := range conflicts {
			if _, exists := closure[conflict]; exists {
				diagnostics = append(diagnostics, diagnostic("capability.conflict", pack.Metadata.Name, fmt.Sprintf("capability %q conflicts with %q", pack.Metadata.Name, conflict)))
			}
		}
	}
	return order, diagnostics
}

func search(catalog Catalog, requirements map[string][]requirement, assigned map[string]spec.CapabilityPack) (map[string]spec.CapabilityPack, *resolutionIssue) {
	for name, pack := range assigned {
		if !satisfiesAll(pack.Metadata.Version, requirements[name]) {
			return nil, unsatisfiedIssue(name, requirements[name], catalog[name])
		}
	}

	var unresolved []string
	for name := range requirements {
		if _, exists := assigned[name]; !exists {
			unresolved = append(unresolved, name)
		}
	}
	if len(unresolved) == 0 {
		return assigned, nil
	}
	sort.Strings(unresolved)
	name := unresolved[0]
	versions, exists := catalog[name]
	if !exists {
		return nil, &resolutionIssue{
			code: "capability.missing", path: name,
			message: fmt.Sprintf("capability %q is not present in the catalog", name),
		}
	}
	candidates := compatibleCandidates(versions, requirements[name])
	if len(candidates) == 0 {
		return nil, unsatisfiedIssue(name, requirements[name], versions)
	}

	var firstIssue *resolutionIssue
	for _, candidate := range candidates {
		nextAssigned := cloneAssignments(assigned)
		nextAssigned[name] = candidate
		nextRequirements := cloneRequirements(requirements)
		for _, dependency := range candidate.Spec.Requires {
			nextRequirements[dependency.Name] = append(nextRequirements[dependency.Name], requirement{
				constraint: dependency.Constraint,
				source:     candidate.Metadata.Name + "@" + candidate.Metadata.Version,
			})
		}
		resolved, issue := search(catalog, nextRequirements, nextAssigned)
		if issue == nil {
			return resolved, nil
		}
		if firstIssue == nil {
			firstIssue = issue
		}
	}
	return nil, firstIssue
}

func compatibleCandidates(versions map[string]spec.CapabilityPack, requirements []requirement) []spec.CapabilityPack {
	candidates := make([]spec.CapabilityPack, 0, len(versions))
	for version, pack := range versions {
		if satisfiesAll(version, requirements) {
			candidates = append(candidates, pack)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		comparison := compareVersions(candidates[i].Metadata.Version, candidates[j].Metadata.Version)
		if comparison == 0 {
			return candidates[i].Metadata.Version > candidates[j].Metadata.Version
		}
		return comparison > 0
	})
	return candidates
}

func satisfiesAll(version string, requirements []requirement) bool {
	for _, required := range requirements {
		if !satisfies(version, required.constraint) {
			return false
		}
	}
	return true
}

func unsatisfiedIssue(name string, requirements []requirement, versions map[string]spec.CapabilityPack) *resolutionIssue {
	constraints := make([]string, 0, len(requirements))
	for _, required := range requirements {
		constraints = append(constraints, fmt.Sprintf("%s (from %s)", required.constraint, required.source))
	}
	sort.Strings(constraints)
	available := make([]string, 0, len(versions))
	for version := range versions {
		available = append(available, version)
	}
	sort.Slice(available, func(i, j int) bool { return compareVersions(available[i], available[j]) > 0 })
	return &resolutionIssue{
		code: "capability.version.unsatisfied", path: name,
		message: fmt.Sprintf("capability %q has no version satisfying [%s]; available versions: [%s]", name, strings.Join(constraints, ", "), strings.Join(available, ", ")),
	}
}

func cloneAssignments(source map[string]spec.CapabilityPack) map[string]spec.CapabilityPack {
	cloned := make(map[string]spec.CapabilityPack, len(source)+1)
	for name, pack := range source {
		cloned[name] = pack
	}
	return cloned
}

func cloneRequirements(source map[string][]requirement) map[string][]requirement {
	cloned := make(map[string][]requirement, len(source)+1)
	for name, requirements := range source {
		cloned[name] = append([]requirement(nil), requirements...)
	}
	return cloned
}

func satisfies(version, constraint string) bool {
	if constraint == "" || constraint == "*" {
		return true
	}
	if strings.HasPrefix(constraint, ">=") {
		return compareVersions(version, strings.TrimPrefix(constraint, ">=")) >= 0
	}
	if strings.HasPrefix(constraint, "^") {
		minimum := strings.TrimPrefix(constraint, "^")
		currentParts, currentOK := coreVersion(version)
		minimumParts, minimumOK := coreVersion(minimum)
		if !currentOK || !minimumOK || compareVersions(version, minimum) < 0 {
			return false
		}
		if minimumParts[0] > 0 {
			return currentParts[0] == minimumParts[0]
		}
		if minimumParts[1] > 0 {
			return currentParts[0] == 0 && currentParts[1] == minimumParts[1]
		}
		return currentParts == minimumParts
	}
	return compareVersions(version, constraint) == 0
}

func compareVersions(left, right string) int {
	leftParts, leftPreRelease, leftOK := parsedVersion(left)
	rightParts, rightPreRelease, rightOK := parsedVersion(right)
	if !leftOK || !rightOK {
		return -1
	}
	for index := range leftParts {
		if leftParts[index] < rightParts[index] {
			return -1
		}
		if leftParts[index] > rightParts[index] {
			return 1
		}
	}
	return comparePreRelease(leftPreRelease, rightPreRelease)
}

func coreVersion(value string) ([3]int, bool) {
	parts, _, ok := parsedVersion(value)
	return parts, ok
}

func parsedVersion(value string) ([3]int, string, bool) {
	value = strings.SplitN(value, "+", 2)[0]
	versionParts := strings.SplitN(value, "-", 2)
	preRelease := ""
	if len(versionParts) == 2 {
		preRelease = versionParts[1]
	}
	parts := strings.Split(versionParts[0], ".")
	if len(parts) != 3 {
		return [3]int{}, "", false
	}
	var result [3]int
	for index, part := range parts {
		parsed, err := strconv.Atoi(part)
		if err != nil || parsed < 0 {
			return [3]int{}, "", false
		}
		result[index] = parsed
	}
	return result, preRelease, true
}

func comparePreRelease(left, right string) int {
	if left == right {
		return 0
	}
	if left == "" {
		return 1
	}
	if right == "" {
		return -1
	}
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	length := min(len(leftParts), len(rightParts))
	for index := 0; index < length; index++ {
		leftNumber, leftNumeric := numericIdentifier(leftParts[index])
		rightNumber, rightNumeric := numericIdentifier(rightParts[index])
		switch {
		case leftNumeric && rightNumeric && leftNumber != rightNumber:
			if leftNumber < rightNumber {
				return -1
			}
			return 1
		case leftNumeric != rightNumeric:
			if leftNumeric {
				return -1
			}
			return 1
		case !leftNumeric && leftParts[index] != rightParts[index]:
			return strings.Compare(leftParts[index], rightParts[index])
		}
	}
	if len(leftParts) < len(rightParts) {
		return -1
	}
	return 1
}

func numericIdentifier(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	return parsed, err == nil
}

func diagnostic(code, path, message string) spec.Diagnostic {
	return spec.Diagnostic{Code: code, Severity: spec.SeverityError, Path: path, Message: message}
}
