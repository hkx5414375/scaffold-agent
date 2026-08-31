// Package capability resolves deterministic capability dependency graphs.
package capability

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

// Resolve returns selected packs and transitive dependencies in dependency-first order.
func Resolve(catalog map[string]spec.CapabilityPack, selected []spec.CapabilitySelection) ([]spec.CapabilityPack, []spec.Diagnostic) {
	var diagnostics []spec.Diagnostic
	requested := make(map[string]string, len(selected))
	for _, selection := range selected {
		if _, exists := requested[selection.Name]; exists {
			diagnostics = append(diagnostics, diagnostic("capability.selection.duplicate", selection.Name, fmt.Sprintf("capability %q is selected more than once", selection.Name)))
			continue
		}
		requested[selection.Name] = selection.Version
	}

	state := make(map[string]uint8, len(catalog))
	var order []spec.CapabilityPack
	var visit func(string, string)
	visit = func(name, constraint string) {
		pack, exists := catalog[name]
		if !exists {
			diagnostics = append(diagnostics, diagnostic("capability.missing", name, fmt.Sprintf("capability %q is not present in the catalog", name)))
			return
		}
		if constraint != "" && !satisfies(pack.Metadata.Version, constraint) {
			diagnostics = append(diagnostics, diagnostic("capability.version.unsatisfied", name, fmt.Sprintf("capability %q version %s does not satisfy %s", name, pack.Metadata.Version, constraint)))
			return
		}
		switch state[name] {
		case 1:
			diagnostics = append(diagnostics, diagnostic("capability.dependency.cycle", name, fmt.Sprintf("capability dependency cycle includes %q", name)))
			return
		case 2:
			return
		}
		state[name] = 1
		dependencies := append([]spec.PackDependency(nil), pack.Spec.Requires...)
		sort.Slice(dependencies, func(i, j int) bool { return dependencies[i].Name < dependencies[j].Name })
		for _, dependency := range dependencies {
			visit(dependency.Name, dependency.Constraint)
		}
		state[name] = 2
		order = append(order, pack)
	}

	roots := make([]string, 0, len(requested))
	for name := range requested {
		roots = append(roots, name)
	}
	sort.Strings(roots)
	for _, name := range roots {
		visit(name, requested[name])
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
