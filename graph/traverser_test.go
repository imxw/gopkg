package graph

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraverser_GetTopologyTree(t *testing.T) {
	// Build a simple graph: A -> B -> C, A -> D (edges are bidirectional)
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	d := uuid.New()

	edges := []Edge{
		{RelationID: uuid.New(), SourceID: a, TargetID: b, RelationType: "contains"},
		{RelationID: uuid.New(), SourceID: b, TargetID: c, RelationType: "contains"},
		{RelationID: uuid.New(), SourceID: a, TargetID: d, RelationType: "connects"},
	}

	tr := NewTraverser(edges)

	tests := []struct {
		name      string
		startID   uuid.UUID
		maxDepth  int
		wantNodes int
		wantEdges int
	}{
		{
			name:      "full tree from A",
			startID:   a,
			maxDepth:  10,
			wantNodes: 4, // A, B, C, D
			wantEdges: 3,
		},
		{
			name:      "partial tree from B (bidirectional: reaches A, C, D)",
			startID:   b,
			maxDepth:  10,
			wantNodes: 4, // B, A, C, D (bidirectional: B→A, B→C, A→D)
			wantEdges: 3,
		},
		{
			name:      "depth 1 from A",
			startID:   a,
			maxDepth:  1,
			wantNodes: 3, // A, B, D
			wantEdges: 2,
		},
		{
			name:      "leaf node C (bidirectional: reaches B, A, D)",
			startID:   c,
			maxDepth:  10,
			wantNodes: 4, // C, B, A, D (bidirectional)
			wantEdges: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, edges := tr.GetTopologyTree(tt.startID, tt.maxDepth)
			assert.Len(t, nodes, tt.wantNodes)
			assert.Len(t, edges, tt.wantEdges)
		})
	}
}

func TestTraverser_GetShortestPath(t *testing.T) {
	// Graph: A -> B -> C -> D, A -> D (direct) — all bidirectional
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	d := uuid.New()

	edges := []Edge{
		{RelationID: uuid.New(), SourceID: a, TargetID: b, RelationType: "contains"},
		{RelationID: uuid.New(), SourceID: b, TargetID: c, RelationType: "contains"},
		{RelationID: uuid.New(), SourceID: c, TargetID: d, RelationType: "contains"},
		{RelationID: uuid.New(), SourceID: a, TargetID: d, RelationType: "connects"},
	}

	tr := NewTraverser(edges)

	tests := []struct {
		name      string
		from      uuid.UUID
		to        uuid.UUID
		wantFound bool
		wantLen   int
	}{
		{
			name:      "direct path A->D",
			from:      a,
			to:        d,
			wantFound: true,
			wantLen:   2, // A -> D (direct)
		},
		{
			name:      "path A->C",
			from:      a,
			to:        c,
			wantFound: true,
			wantLen:   3, // A -> B -> C
		},
		{
			name:      "reverse path D->A (bidirectional)",
			from:      d,
			to:        a,
			wantFound: true,
			wantLen:   2, // D -> A (direct, via reverse edge)
		},
		{
			name:      "path D->B (bidirectional)",
			from:      d,
			to:        b,
			wantFound: true,
			wantLen:   3, // D -> A -> B
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tr.GetShortestPath(tt.from, tt.to, 0)
			if !tt.wantFound {
				assert.Nil(t, path)
				return
			}
			require.NotNil(t, path)
			assert.Len(t, path.Nodes, tt.wantLen)
			assert.Equal(t, tt.from, path.Nodes[0])
			assert.Equal(t, tt.to, path.Nodes[len(path.Nodes)-1])
		})
	}
}

func TestTraverser_GetImpactAnalysis(t *testing.T) {
	// Graph: A -> B -> C -> D (impact analysis is forward-only)
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	d := uuid.New()

	edges := []Edge{
		{RelationID: uuid.New(), SourceID: a, TargetID: b, RelationType: "depends_on"},
		{RelationID: uuid.New(), SourceID: b, TargetID: c, RelationType: "depends_on"},
		{RelationID: uuid.New(), SourceID: c, TargetID: d, RelationType: "depends_on"},
	}

	tr := NewTraverser(edges)

	tests := []struct {
		name      string
		startID   uuid.UUID
		maxDepth  int
		wantNodes int
	}{
		{
			name:      "impact from A",
			startID:   a,
			maxDepth:  10,
			wantNodes: 4, // A -> B -> C -> D
		},
		{
			name:      "impact from B (forward only: C, D)",
			startID:   b,
			maxDepth:  10,
			wantNodes: 3, // B -> C -> D
		},
		{
			name:      "leaf D has no downstream impact",
			startID:   d,
			maxDepth:  10,
			wantNodes: 1, // D only
		},
		{
			name:      "limited depth",
			startID:   a,
			maxDepth:  2,
			wantNodes: 3, // A -> B -> C
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, _ := tr.GetImpactAnalysis(tt.startID, tt.maxDepth)
			assert.Len(t, nodes, tt.wantNodes)
		})
	}
}

func TestTraverser_CycleDetection(t *testing.T) {
	// Graph with cycle: A -> B -> C -> A
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()

	edges := []Edge{
		{RelationID: uuid.New(), SourceID: a, TargetID: b, RelationType: "contains"},
		{RelationID: uuid.New(), SourceID: b, TargetID: c, RelationType: "contains"},
		{RelationID: uuid.New(), SourceID: c, TargetID: a, RelationType: "contains"},
	}

	tr := NewTraverser(edges)

	// Should not infinite loop
	nodes, edgesOut := tr.GetTopologyTree(a, 10)
	assert.Len(t, nodes, 3) // A, B, C (each visited once)
	assert.Len(t, edgesOut, 2) // A->B, B->C (C->A skipped, A already visited)
}

func TestTraverser_DisconnectedComponents(t *testing.T) {
	// Two disconnected subgraphs: A-B and C-D
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	d := uuid.New()

	edges := []Edge{
		{RelationID: uuid.New(), SourceID: a, TargetID: b, RelationType: "contains"},
		{RelationID: uuid.New(), SourceID: c, TargetID: d, RelationType: "contains"},
	}

	tr := NewTraverser(edges)

	// Shortest path from A to D should be nil (no connection)
	path := tr.GetShortestPath(a, d, 0)
	assert.Nil(t, path)

	// BFS from A should only reach B
	nodes, edgesOut := tr.BFS(a, 10)
	assert.Len(t, nodes, 2) // A, B
	assert.Len(t, edgesOut, 1)

	// Shortest path A->B should work
	path = tr.GetShortestPath(a, b, 0)
	require.NotNil(t, path)
	assert.Len(t, path.Nodes, 2)
}
