package graphs

import (
	"fmt"
	"testing"

	types "ced.io/jobgraphs/pkg/apis/v1alpha1"
)

func compare(arr1 []string, arr2 []string) bool {
	m := make(map[string]bool)

	for i := 0; i < len(arr1); i++ {
		m[arr1[i]] = true
	}
	for i := 0; i < len(arr2); i++ {
		if _, ok := m[arr2[i]]; !ok {
			return false
		}
	}
	return true
}

func TestOutgoing(t *testing.T) {

	graphSpec := types.GraphSpec{
		Edges: []types.Edge{{Source: "n1", Target: "n2"}, {Source: "n1", Target: "n3"}, {Source: "n2", Target: "n4"}},
	}

	outgoing := Outgoing(&graphSpec, "n1")

	if !compare(outgoing, []string{"n2", "n3"}) {
		t.Error("")
	}

	outgoing = Outgoing(&graphSpec, "n4")

	if !compare(outgoing, []string{}) {
		t.Error("")
	}
}

func TestIncoming(t *testing.T) {
	graphSpec := types.GraphSpec{
		Edges: []types.Edge{
			{Source: "n1", Target: "n2"},
			{Source: "n1", Target: "n3"},
			{Source: "n2", Target: "n4"},
			{Source: "n3", Target: "n4"},
		},
	}
	incoming := Incoming(&graphSpec, "n4")

	if !compare(incoming, []string{"n2", "n3"}) {
		t.Error("")
	}
	incoming = Incoming(&graphSpec, "n2")

	if !compare(incoming, []string{"n1"}) {
		t.Error("")
	}
	incoming = Incoming(&graphSpec, "n1")

	if !compare(incoming, []string{}) {
		t.Error("")
	}
}

func TestRoots(t *testing.T) {
	graphSpec := types.GraphSpec{
		Edges: []types.Edge{
			{Source: "n1", Target: "n2"},
			{Source: "n1", Target: "n3"},
			{Source: "n2", Target: "n4"},
			{Source: "n3", Target: "n4"},
		},
	}

	roots := Roots(&graphSpec)

	if !compare(roots, []string{"n1"}) {
		t.Error()
	}
	graphSpec = types.GraphSpec{
		Edges: []types.Edge{
			{Source: "n1", Target: "n2"},
			{Source: "n1", Target: "n3"},
			{Source: "n2", Target: "n4"},
			{Source: "n3", Target: "n4"},
			{Source: "n5", Target: "n2"},
		},
	}

	roots = Roots(&graphSpec)

	if !compare(roots, []string{"n1", "n5"}) {
		t.Error()
	}
}

func TestLeaves(t *testing.T) {
	graphSpec := types.GraphSpec{
		Edges: []types.Edge{
			{Source: "n1", Target: "n2"},
			{Source: "n1", Target: "n3"},
			{Source: "n2", Target: "n4"},
			{Source: "n3", Target: "n4"},
		},
	}

	leaves := Leaves(&graphSpec)

	if !compare(leaves, []string{"n4"}) {
		t.Error()
	}
	graphSpec = types.GraphSpec{
		Edges: []types.Edge{
			{Source: "n1", Target: "n2"},
			{Source: "n1", Target: "n3"},
			{Source: "n2", Target: "n4"},
			{Source: "n3", Target: "n4"},
			{Source: "n3", Target: "n5"},
		},
	}

	leaves = Leaves(&graphSpec)

	if !compare(leaves, []string{"n4", "n5"}) {
		t.Error()
	}
}

func TestCheckAcyclic(t *testing.T) {
	graphSpec := types.GraphSpec{
		Edges: []types.Edge{
			{Source: "n1", Target: "n2"},
			{Source: "n2", Target: "n1"},
		},
	}

	err := CheckAcyclic(&graphSpec)

	if err == nil {
		t.Error("Should be cyclic")
	}
	graphSpec = types.GraphSpec{
		Edges: []types.Edge{
			{Source: "n1", Target: "n2"},
			{Source: "n2", Target: "n3"},
			{Source: "n3", Target: "n2"},
		},
	}

	err = CheckAcyclic(&graphSpec)

	if err == nil {
		t.Error("Should be cyclic")
	}
	fmt.Printf("********************")
	graphSpec = types.GraphSpec{
		Edges: []types.Edge{
			{Source: "n1", Target: "n2"},
			{Source: "n2", Target: "n4"},
			{Source: "n1", Target: "n3"},
			{Source: "n3", Target: "n4"},
		},
	}

	err = CheckAcyclic(&graphSpec)

	if err != nil {
		t.Error("Should be acyclic", err)
	}
}
