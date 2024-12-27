package unfolder

import (
    "errors"
    "fmt"
    "math"
)

// -----------------------------
//  Basic Data Structures
// -----------------------------

// Vector3 represents a 3D point or vector.
type Vector3 struct {
    X, Y, Z float64
}

// Point2 represents a 2D point.
type Point2 struct {
    X, Y float64
}

// Face holds indices to vertices in the Polyhedron (in CCW order).
type Face struct {
    Vertices []int
}

// Polyhedron holds the 3D model data: a set of vertices and faces.
type Polyhedron struct {
    Vertices []Vector3
    Faces    []Face
    Name     string
}

// Adjacency info: for each face, which other faces are adjacent and by which edge?
type FaceNeighbor struct {
    FaceIndex     int   // index of the adjacent face
    SharedEdge    [2]int  // the vertex indices of the shared edge
    ThisFaceEdge  [2]int  // which edge in "this" face corresponds to SharedEdge
}

// FaceAdjacency stores adjacency for each face: a list of neighbors
type FaceAdjacency struct {
    Neighbors map[int][]FaceNeighbor
}

// -----------------------------
//   1) Build Face Adjacency
// -----------------------------

// BuildFaceAdjacency finds which faces share edges. We assume manifold geometry:
// each edge belongs to either 1 or 2 faces.
func BuildFaceAdjacency(poly Polyhedron) (*FaceAdjacency, error) {
    nFaces := len(poly.Faces)
    adj := FaceAdjacency{
        Neighbors: make(map[int][]FaceNeighbor, nFaces),
    }
    
    // Edge map: key = (minVertex, maxVertex), value = []faceIndex
    edgeMap := make(map[[2]int][]int)

    // Populate edgeMap
    for fIdx, face := range poly.Faces {
        vCount := len(face.Vertices)
        for i := 0; i < vCount; i++ {
            vA := face.Vertices[i]
            vB := face.Vertices[(i+1)%vCount]
            edge := sortPair(vA, vB)
            
            edgeMap[edge] = append(edgeMap[edge], fIdx)
        }
    }

    // Now, each entry in edgeMap with 2 faces means those faces share that edge
    for edge, faceList := range edgeMap {
        if len(faceList) == 2 {
            f0 := faceList[0]
            f1 := faceList[1]

            // figure out the local edge indices in each face
            face0Edge, err0 := findEdgeInFace(poly.Faces[f0], edge)
            face1Edge, err1 := findEdgeInFace(poly.Faces[f1], edge)
            if err0 != nil || err1 != nil {
                // Should not happen if geometry is consistent
                continue
            }

            // add adjacency entry for face0
            adj.Neighbors[f0] = append(adj.Neighbors[f0], FaceNeighbor{
                FaceIndex:    f1,
                SharedEdge:   edge,
                ThisFaceEdge: face0Edge,
            })
            // add adjacency entry for face1
            adj.Neighbors[f1] = append(adj.Neighbors[f1], FaceNeighbor{
                FaceIndex:    f0,
                SharedEdge:   edge,
                ThisFaceEdge: face1Edge,
            })
        }
    }

    return &adj, nil
}

// sortPair returns a 2-int array with the smaller one first
func sortPair(a, b int) [2]int {
    if a < b {
        return [2]int{a, b}
    }
    return [2]int{b, a}
}

// findEdgeInFace returns the consecutive vertex indices in the face that correspond
// to the sorted edge. Example: if the face has vertices [A,B,C,D], and the edge is (A,B),
// then we return something like [0,1].
func findEdgeInFace(face Face, sortedEdge [2]int) ([2]int, error) {
    vCount := len(face.Vertices)
    for i := 0; i < vCount; i++ {
        j := (i + 1) % vCount
        edgeIJ := sortPair(face.Vertices[i], face.Vertices[j])
        if edgeIJ == sortedEdge {
            return [2]int{i, j}, nil
        }
    }
    return [2]int{-1, -1}, errors.New("edge not found in face")
}

// -----------------------------
//   2) Pick a Spanning Tree
// -----------------------------

// BuildFaceSpanningTree uses BFS starting from face 0 (or any rootFace) to pick edges to "cut".
// Returns an array `parent` of length nFaces, where parent[i] = -1 if i is root, or the face
// that discovered i in BFS. This effectively forms a spanning tree in the face graph.
func BuildFaceSpanningTree(adj *FaceAdjacency, rootFace int, nFaces int) []int {
    parent := make([]int, nFaces)
    for i := 0; i < nFaces; i++ {
        parent[i] = -1
    }

    visited := make([]bool, nFaces)
    queue := []int{rootFace}
    visited[rootFace] = true

    // BFS
    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]

        neighbors := adj.Neighbors[current]
        for _, nbr := range neighbors {
            if !visited[nbr.FaceIndex] {
                visited[nbr.FaceIndex] = true
                parent[nbr.FaceIndex] = current
                queue = append(queue, nbr.FaceIndex)
            }
        }
    }

    return parent
}

// -----------------------------
//   3) Face-by-Face Unfolding
// -----------------------------

// We store for each face a 2D transform (rotation + translation) or we can store
// the 2D coordinates of that face's vertices. We'll do the latter for simplicity.

type Face2D struct {
    Vertices []Point2 // 2D coordinates of each vertex of this face
}

// UnfoldResult holds the final 2D positions for each vertex in the mesh
// plus face-level info. You can also store "cuts" if needed.
type UnfoldResult struct {
    Vertex2D  []Point2
    Face2D    []Face2D
    SpanningTree []int // parent array from BFS
}

// UnfoldMesh flattens the polyhedron into a single connected net, ignoring overlaps.
// - rootFace is the index of the face we place first in 2D
func UnfoldMesh(poly Polyhedron, rootFace int) (*UnfoldResult, error) {
    if len(poly.Faces) == 0 {
        return nil, errors.New("polyhedron has no faces")
    }

    // 1) Build adjacency
    adjacency, err := BuildFaceAdjacency(poly)
    if err != nil {
        return nil, fmt.Errorf("error building adjacency: %v", err)
    }

    nFaces := len(poly.Faces)
    nVerts := len(poly.Vertices)

    // 2) BFS spanning tree (which edges are "cuts")
    parent := BuildFaceSpanningTree(adjacency, rootFace, nFaces)

    // We'll keep track of whether each face is "placed" in 2D
    placed := make([]bool, nFaces)
    placed[rootFace] = true

    // Face2D array
    face2Ds := make([]Face2D, nFaces)
    // We'll also store a global 2D coordinate for each vertex
    // (This is simplistic: if a vertex belongs to multiple faces,
    // they might not coincide if there's overlap or non-developable geometry,
    // but let's ignore that for now.)
    vertex2D := make([]Point2, nVerts)

    // 3) Place the root face in 2D
    err = placeRootFace(poly, rootFace, &face2Ds[rootFace], vertex2D)
    if err != nil {
        return nil, fmt.Errorf("failed to place root face: %v", err)
    }

    // BFS queue
    queue := []int{rootFace}

    for len(queue) > 0 {
        fIdx := queue[0]
        queue = queue[1:]

        // For each adjacent face
        for _, nbr := range adjacency.Neighbors[fIdx] {
            nfIdx := nbr.FaceIndex
            // If that face's parent is the current face => this is the BFS tree edge
            if parent[nfIdx] == fIdx && !placed[nfIdx] {
                // place neighbor face in 2D
                err = placeAdjacentFace(poly, fIdx, nfIdx, &face2Ds[nfIdx], vertex2D, &nbr)
                if err != nil {
                    return nil, fmt.Errorf("failed to place face %d adjacent to %d: %v", nfIdx, fIdx, err)
                }
                placed[nfIdx] = true
                queue = append(queue, nfIdx)
            }
        }
    }

    return &UnfoldResult{
        Vertex2D:     vertex2D,
        Face2D:       face2Ds,
        SpanningTree: parent,
    }, nil
}

// placeRootFace simply puts the root face in the plane so that:
// - The first vertex is at (0,0)
// - The second vertex is at (edgeLength, 0)
// - The rest of the vertices are placed accordingly in the plane of the face
func placeRootFace(poly Polyhedron, faceIdx int, face2D *Face2D, vertex2D []Point2) error {
    face := poly.Faces[faceIdx]
    vCount := len(face.Vertices)
    if vCount < 3 {
        return errors.New("face has fewer than 3 vertices")
    }

    // Let's define a local 3D coordinate system for this face:
    //   - origin at the first vertex
    //   - x-axis along the edge from first to second vertex
    //   - y-axis in the plane of the face
    // This gives us a 3x3 rotation matrix (or we can do it ad-hoc with vector math).

    // 1) get 3D points
    p0 := poly.Vertices[face.Vertices[0]]
    p1 := poly.Vertices[face.Vertices[1]]
    p2 := poly.Vertices[face.Vertices[2]]

    // 2) define the plane coordinate axes
    e01 := sub(p1, p0) // first edge
    e02 := sub(p2, p0)

    xAxis := normalize(e01)
    // yAxis is in the plane: cross the face normal with xAxis or something similar
    normal := cross(e01, e02)
    normal = normalize(normal)
    yAxis := cros
