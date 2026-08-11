package testsupport

import "sort"

type RuntimeLeakBaseline struct {
	Namespaces        []string
	Interfaces        []string
	BridgeMemberships []string
	Routes            []string
	Rules             []string
	Sockets           []string
	Processes         []string
	Artifacts         []string
}

type RuntimeLeakDiff struct {
	Namespaces        []string
	Interfaces        []string
	BridgeMemberships []string
	Routes            []string
	Rules             []string
	Sockets           []string
	Processes         []string
	Artifacts         []string
}

func (baseline RuntimeLeakBaseline) Diff(final RuntimeLeakBaseline) RuntimeLeakDiff {
	return RuntimeLeakDiff{
		Namespaces:        addedValues(baseline.Namespaces, final.Namespaces),
		Interfaces:        addedValues(baseline.Interfaces, final.Interfaces),
		BridgeMemberships: addedValues(baseline.BridgeMemberships, final.BridgeMemberships),
		Routes:            addedValues(baseline.Routes, final.Routes),
		Rules:             addedValues(baseline.Rules, final.Rules),
		Sockets:           addedValues(baseline.Sockets, final.Sockets),
		Processes:         addedValues(baseline.Processes, final.Processes),
		Artifacts:         addedValues(baseline.Artifacts, final.Artifacts),
	}
}

func (diff RuntimeLeakDiff) Empty() bool {
	return len(diff.Namespaces) == 0 && len(diff.Interfaces) == 0 && len(diff.BridgeMemberships) == 0 && len(diff.Routes) == 0 && len(diff.Rules) == 0 && len(diff.Sockets) == 0 && len(diff.Processes) == 0 && len(diff.Artifacts) == 0
}

func addedValues(before, after []string) []string {
	known := make(map[string]struct{}, len(before))
	for _, value := range before {
		known[value] = struct{}{}
	}
	result := make([]string, 0)
	for _, value := range after {
		if _, exists := known[value]; !exists {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
