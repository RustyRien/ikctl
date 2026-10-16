package registry

import "testing"

func TestLookup(t *testing.T) {
	tests := map[string]string{
		"resource": "resource", "Resources": "resource", "rs": "resource",
		"scv": "source-code-version", "sourceCodeVersions": "source-code-version",
		"source_code_version": "source-code-version", "SourceCode": "source-code",
		"audit-logs": "audit-log", "wf": "workflow",
	}
	for in, want := range tests {
		k, ok := Lookup(in)
		if !ok || k.Name != want {
			t.Errorf("Lookup(%q) = %q, %v; want %q", in, k.Name, ok, want)
		}
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("Lookup(nope) should fail")
	}
}

func TestOperationNames(t *testing.T) {
	k, _ := Lookup("scv")
	got := []string{k.GetOp(), k.ListOp(), k.CountOp(), k.ActionsOp(), k.CreateOp(), k.UpdateOp(), k.ActionOp(), k.DeleteOp(), k.CreateInputType()}
	want := []string{"sourceCodeVersion", "sourceCodeVersions", "sourceCodeVersionsCount", "sourceCodeVersionActions",
		"createSourceCodeVersion", "updateSourceCodeVersion", "sourceCodeVersionAction", "deleteSourceCodeVersion",
		"SourceCodeVersionCreateInput"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("op %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestKindsAreConsistent(t *testing.T) {
	seen := map[string]string{}
	for _, k := range All() {
		if k.Name == "" || k.Plural == "" || k.Title == "" || k.GraphQLName == "" || k.TypeName == "" || k.IDType == "" {
			t.Errorf("kind %q has empty required fields", k.Name)
		}
		if k.ListFields == "" || k.DescribeFields == "" || len(k.Columns) == 0 || len(k.Verbs) == 0 {
			t.Errorf("kind %q has no fields, columns or verbs", k.Name)
		}
		for _, n := range append([]string{k.Name, k.Plural}, k.Aliases...) {
			if other, dup := seen[n]; dup {
				t.Errorf("name %q used by %q and %q", n, other, k.Name)
			}
			seen[n] = k.Name
		}
		if k.NamePath != "" && k.NameFilter == "" {
			t.Errorf("kind %q has NamePath without NameFilter", k.Name)
		}
	}
}
