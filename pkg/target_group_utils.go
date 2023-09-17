package gmakec

import "golang.org/x/exp/slices"

// Don't ask me why this works.
// This "algorithm" came about when I was trying to create the dependency graph.
// Maybe there's some maths formulae or algorithms which would improve this code.
// However, it works at the moment and is actually not slow (at least on my machine).
//
// Basically it figures out which targets to build first and puts them in a "matrix".
// Example:
//
//	Target index 0: no dependencies
//	Target index 1: dependency on target 0
//	Target index 2: no dependencies
//	Target index 3: dependency on target 1
//
// Would result in: [[0, 1, 3] [2]]
// Targets will be built in exactly the order of the two target groups (inner arrays).
// The target groups will also be built in parallel.
//
// I'm always open for suggestions. :)
// ~ thetredev
func generateTargetGroupMatrix(graphs [][]int) [][]int {
	targetGroupMatrix := [][]int{}

	for i := len(graphs) - 1; i >= 0; i-- {
		graph := graphs[i]
		mergedGraph := graph

		for _, graphIndex := range graph {
			for j, outerGraph := range graphs {
				if i == j {
					break
				}

				found := false

				for _, outerGraphIndex := range outerGraph {
					toAdd := -1

					if graphIndex == outerGraphIndex {
						found = true
						toAdd = graphIndex
					} else if found {
						toAdd = outerGraphIndex
					}

					if toAdd > -1 && !slices.Contains(mergedGraph, toAdd) {
						mergedGraph = append(mergedGraph, toAdd)
					}
				}
			}
		}

		if slices.Compare(graph, mergedGraph) != 0 {
			targetGroupMatrix = append(targetGroupMatrix, mergedGraph)
		} else {
			isRemainder := false

			for _, sortedGraphItem := range targetGroupMatrix {
				for _, mergedGraphIndex := range mergedGraph {
					if slices.Contains(sortedGraphItem, mergedGraphIndex) {
						isRemainder = true
						break
					}
				}

				if isRemainder {
					break
				}
			}

			if !isRemainder {
				targetGroupMatrix = append(targetGroupMatrix, mergedGraph)
			}
		}
	}

	return targetGroupMatrix
}
