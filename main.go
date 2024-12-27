package main

import (
    "fmt"
    "log"

    "github.com/yourusername/unfolder" // Adjust to your module path
)

func main() {
    // Example: build a simple cube
    poly := buildUnitCube()
    // Unfold from face 0 as root
    result, err := unfolder.UnfoldMesh(poly, 0)
    if err != nil {
        log.Fatalf("Unfold failed: %v\n", err)
    }

    fmt.Printf("Spanning tree (parent array) = %v\n", result.SpanningTree)

    // Print 2D coords for all vertices
    for i, v2 := range result.Vertex2D {
        fmt.Printf("Vertex %d => (%.3f, %.3f)\n", i, v2.X, v2.Y)
    }

    // Each face’s local 2D coords
    for fIdx, f2d := range result.Face2D {
        fmt.Printf("Face %d => ", fIdx)
        for _, p := range f2d.Vertices {
            fmt.Printf("(%.2f, %.2f) ", p.X, p.Y)
        }
        fmt.Println()
    }
}

// buildUnitCube returns a Polyhedron for a unit cube (side=1) with
// 8 vertices at [0 or 1, 0 or 1, 0 or 1], 6 faces.
func buildUnitCube() unfolder.Polyhedron {
    verts := []unfolder.Vector3{
        {0, 0, 0}, // 0
        {1, 0, 0}, // 1
        {1, 1, 0}, // 2
        {0, 1, 0}, // 3
        {0, 0, 1}, // 4
        {1, 0, 1}, // 5
        {1, 1, 1}, // 6
        {0, 1, 1}, // 7
    }
    // Each face as a loop of vertex indices (CCW order)
    faces := []unfolder.Face{
        {Vertices: []int{0, 1, 2, 3}}, // bottom
        {Vertices: []int{4, 5, 6, 7}}, // top
        {Vertices: []int{0, 1, 5, 4}}, // front
        {Vertices: []int{1, 2, 6, 5}}, // right
        {Vertices: []int{2, 3, 7, 6}}, // back
        {Vertices: []int{3, 0, 4, 7}}, // left
    }

    return unfolder.Polyhedron{
        Vertices: verts,
        Faces:    faces,
        Name:     "UnitCube",
    }
}
