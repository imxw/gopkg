// Package graph provides generic graph traversal algorithms including BFS and shortest path.
package graph

import "github.com/google/uuid"

// Edge represents a directed edge in the CI relationship graph.
type Edge struct {
	RelationID   uuid.UUID
	SourceID     uuid.UUID
	TargetID     uuid.UUID
	RelationType string
}

// Path represents a path between two nodes in the graph.
type Path struct {
	Nodes []uuid.UUID
	Edges []Edge
}

// NodeInfo represents a node with depth information for topology traversal.
type NodeInfo struct {
	ID    uuid.UUID
	Depth int
}

// EdgeInfo represents an edge with depth information.
type EdgeInfo struct {
	Edge  Edge
	Depth int
}

// parentInfo tracks BFS parent pointers for path backtracking.
type parentInfo struct {
	node uuid.UUID
	edge Edge
}

// bfsQueueItem is an entry in the BFS queue storing the current node and its depth.
type bfsQueueItem struct {
	id    uuid.UUID
	depth int
}

// neighbor represents a reachable neighbor with the edge that leads to it.
type neighbor struct {
	node uuid.UUID
	edge Edge
}

// Traverser provides BFS-based graph traversal for CI topology.
type Traverser struct {
	edges []Edge
	// adj stores both forward and reverse neighbors for undirected traversal.
	adj map[uuid.UUID][]neighbor
	// fwdAdj stores only forward neighbors for directed traversal (impact analysis).
	fwdAdj map[uuid.UUID][]neighbor
}

// NewTraverser creates a new Traverser from a set of edges.
// Builds bidirectional adjacency: each edge is traversable in both directions.
func NewTraverser(edges []Edge) *Traverser {
	adj := make(map[uuid.UUID][]neighbor, len(edges)*2)
	fwdAdj := make(map[uuid.UUID][]neighbor, len(edges))
	for _, e := range edges {
		// Forward: source → target
		adj[e.SourceID] = append(adj[e.SourceID], neighbor{node: e.TargetID, edge: e})
		// Reverse: target → source (swap direction to match traversal)
		revEdge := Edge{
			RelationID:   e.RelationID,
			SourceID:     e.TargetID,
			TargetID:     e.SourceID,
			RelationType: e.RelationType,
		}
		adj[e.TargetID] = append(adj[e.TargetID], neighbor{node: e.SourceID, edge: revEdge})
		// Forward-only adjacency for directed traversal
		fwdAdj[e.SourceID] = append(fwdAdj[e.SourceID], neighbor{node: e.TargetID, edge: e})
	}
	return &Traverser{edges: edges, adj: adj, fwdAdj: fwdAdj}
}

// BFS performs BFS from startID and returns all reachable nodes and edges.
// Traverses edges bidirectionally (undirected graph behavior).
func (t *Traverser) BFS(startID uuid.UUID, maxDepth int) ([]NodeInfo, []EdgeInfo) {
	visited := map[uuid.UUID]bool{startID: true}
	nodes := make([]NodeInfo, 0, 16)
	edges := make([]EdgeInfo, 0, 16)

	queue := make([]NodeInfo, 0, 64)
	head := 0
	queue = append(queue, NodeInfo{ID: startID, Depth: 0})
	nodes = append(nodes, queue[0])

	for head < len(queue) {
		current := queue[head]
		head++

		if maxDepth > 0 && current.Depth >= maxDepth {
			continue
		}

		for _, n := range t.adj[current.ID] {
			if visited[n.node] {
				continue
			}
			visited[n.node] = true
			next := NodeInfo{ID: n.node, Depth: current.Depth + 1}
			nodes = append(nodes, next)
			edges = append(edges, EdgeInfo{Edge: n.edge, Depth: next.Depth})
			queue = append(queue, next)
		}
	}

	return nodes, edges
}

// GetTopologyTree is an alias for BFS.
func (t *Traverser) GetTopologyTree(startID uuid.UUID, maxDepth int) ([]NodeInfo, []EdgeInfo) {
	return t.BFS(startID, maxDepth)
}

// GetImpactAnalysis performs forward-only BFS (downstream impact).
func (t *Traverser) GetImpactAnalysis(startID uuid.UUID, maxDepth int) ([]NodeInfo, []EdgeInfo) {
	visited := map[uuid.UUID]bool{startID: true}
	nodes := make([]NodeInfo, 0, 16)
	edges := make([]EdgeInfo, 0, 16)

	queue := make([]NodeInfo, 0, 64)
	head := 0
	queue = append(queue, NodeInfo{ID: startID, Depth: 0})
	nodes = append(nodes, queue[0])

	for head < len(queue) {
		current := queue[head]
		head++

		if maxDepth > 0 && current.Depth >= maxDepth {
			continue
		}

		for _, n := range t.fwdAdj[current.ID] {
			if visited[n.node] {
				continue
			}
			visited[n.node] = true
			next := NodeInfo{ID: n.node, Depth: current.Depth + 1}
			nodes = append(nodes, next)
			edges = append(edges, EdgeInfo{Edge: n.edge, Depth: next.Depth})
			queue = append(queue, next)
		}
	}

	return nodes, edges
}

// GetShortestPath finds the shortest path from fromID to toID using BFS.
// Traverses edges bidirectionally (undirected graph behavior).
// maxDepth limits the search depth; pass 0 or negative for unlimited.
// Returns nil if no path exists.
func (t *Traverser) GetShortestPath(fromID, toID uuid.UUID, maxDepth int) *Path {
	if fromID == toID {
		return &Path{Nodes: []uuid.UUID{fromID}}
	}

	// Use parent-pointer BFS for O(n) backtracking instead of O(n²) path copying.
	parent := make(map[uuid.UUID]parentInfo)
	visited := map[uuid.UUID]bool{fromID: true}
	queue := make([]bfsQueueItem, 0, 64)
	head := 0
	queue = append(queue, bfsQueueItem{id: fromID, depth: 0})

	for head < len(queue) {
		current := queue[head]
		head++

		if maxDepth > 0 && current.depth >= maxDepth {
			continue
		}

		for _, n := range t.adj[current.id] {
			if visited[n.node] {
				continue
			}
			visited[n.node] = true
			parent[n.node] = parentInfo{node: current.id, edge: n.edge}

			if n.node == toID {
				return backtrackPath(fromID, toID, parent)
			}

			queue = append(queue, bfsQueueItem{id: n.node, depth: current.depth + 1})
		}
	}

	return nil
}

func backtrackPath(fromID, toID uuid.UUID, parent map[uuid.UUID]parentInfo) *Path {
	// Walk back from target to source
	var nodePath []uuid.UUID
	var edgePath []Edge
	cur := toID
	for cur != fromID {
		p := parent[cur]
		nodePath = append(nodePath, cur)
		edgePath = append(edgePath, p.edge)
		cur = p.node
	}
	nodePath = append(nodePath, fromID)

	// Reverse both slices
	for i, j := 0, len(nodePath)-1; i < j; i, j = i+1, j-1 {
		nodePath[i], nodePath[j] = nodePath[j], nodePath[i]
	}
	for i, j := 0, len(edgePath)-1; i < j; i, j = i+1, j-1 {
		edgePath[i], edgePath[j] = edgePath[j], edgePath[i]
	}

	return &Path{Nodes: nodePath, Edges: edgePath}
}
