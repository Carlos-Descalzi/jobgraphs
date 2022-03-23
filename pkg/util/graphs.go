package graphs

import (
	"fmt"

	types "ced.io/jobgraphs/pkg/apis/v1alpha1"
)

func NodeCount(g *types.GraphSpec) int {
	nodes := make(map[string]bool)

	for i := 0; i < len(g.Edges); i++ {
		nodes[g.Edges[i].Source] = true
		nodes[g.Edges[i].Target] = true
	}

	return len(nodes)
}

func Outgoing(g *types.GraphSpec, node string) []string {

	var outgoing []string

	for i := 0; i < len(g.Edges); i++ {
		if g.Edges[i].Source == node {
			outgoing = append(outgoing, g.Edges[i].Target)
		}
	}

	return outgoing
}

func Incoming(g *types.GraphSpec, node string) []string {
	var incoming []string

	for i := 0; i < len(g.Edges); i++ {
		if g.Edges[i].Target == node {
			incoming = append(incoming, g.Edges[i].Source)
		}
	}

	return incoming
}

func Roots(g *types.GraphSpec) []string {

	roots := make(map[string]bool)

	l := len(g.Edges)

	for i := 0; i < l; i++ {
		var root = true
		for j := 0; j < l; j++ {
			if g.Edges[j].Target == g.Edges[i].Source {
				root = false
				break
			}
		}
		if root {
			roots[g.Edges[i].Source] = true
		}
	}

	result := make([]string, 0, len(roots))

	for k := range roots {
		result = append(result, k)
	}

	return result
}

func Leaves(g *types.GraphSpec) []string {
	leaves := make(map[string]bool)

	l := len(g.Edges)

	for i := 0; i < l; i++ {
		var leaf = true
		for j := 0; j < l; j++ {
			if g.Edges[j].Source == g.Edges[i].Target {
				leaf = false
				break
			}
		}
		if leaf {
			leaves[g.Edges[i].Target] = true
		}
	}

	result := make([]string, 0, len(leaves))
	for k := range leaves {
		result = append(result, k)
	}

	return result
}

/**
Check that the input graph is acyclic
**/
func CheckAcyclic(g *types.GraphSpec) error {

	roots := Roots(g)
	leaves := Leaves(g)

	if len(roots) == 0 || len(leaves) == 0 {
		// no roots mean cyclic
		return fmt.Errorf("Circular reference")
	}

	leavesMap := make(map[string]bool)
	for i := 0; i < len(leaves); i++ {
		leavesMap[leaves[i]] = true
	}

	for i := 0; i < len(roots); i++ {
		path := make(map[string]bool)
		if !checkPath(g, roots[i], leavesMap, path) {
			return fmt.Errorf("Circular reference")
		}
	}

	return nil
}

func checkPath(g *types.GraphSpec, source string, leavesMap map[string]bool, path map[string]bool) bool {

	if _, ok := leavesMap[source]; ok {
		return true // it's a leaf
	}

	if _, ok := path[source]; ok {
		return false // already walked through node
	}
	path[source] = true

	outgoing := Outgoing(g, source)
	for i := 0; i < len(outgoing); i++ {
		if !checkPath(g, outgoing[i], leavesMap, path) {
			return false
		}
	}
	return true
}
