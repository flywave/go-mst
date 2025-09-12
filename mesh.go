package mst

import (
	"math"

	dmat "github.com/flywave/go3d/float64/mat4"
	dvec3 "github.com/flywave/go3d/float64/vec3"

	"github.com/flywave/go3d/vec2"
	"github.com/flywave/go3d/vec3"
)

type MeshNode struct {
	Vertices  []vec3.T        `json:"vertices"`
	Normals   []vec3.T        `json:"normals,omitempty"`
	Colors    [][3]byte       `json:"colors,omitempty"`
	TexCoords []vec2.T        `json:"texCoords,omitempty"`
	Mat       *dmat.T         `json:"mat,omitempty"`
	FaceGroup []*MeshTriangle `json:"faceGroup,omitempty"`
	EdgeGroup []*MeshOutline  `json:"edgeGroup,omitempty"`
	Props     *Properties     `json:"props,omitempty"`
}

func (n *MeshNode) ResortVtVn(m *Mesh) {
	var vs, vns []vec3.T
	var vts []vec2.T
	var idx uint32
	for _, g := range n.FaceGroup {
		for _, f := range g.Faces {
			if f.Normal != nil {
				vns = append(vns, n.Normals[int((*f.Normal)[0])])
				vns = append(vns, n.Normals[int((*f.Normal)[1])])
				vns = append(vns, n.Normals[int((*f.Normal)[2])])
			} else {
				vns = append(vns, vec3.T{0, 0, 1})
				vns = append(vns, vec3.T{0, 0, 1})
				vns = append(vns, vec3.T{0, 0, 1})
			}
			if f.Uv != nil {
				vts = append(vts, n.TexCoords[int((*f.Uv)[0])])
				vts = append(vts, n.TexCoords[int((*f.Uv)[1])])
				vts = append(vts, n.TexCoords[int((*f.Uv)[2])])
			} else {
				vts = append(vts, vec2.T{0, 0})
				vts = append(vts, vec2.T{0, 0})
				vts = append(vts, vec2.T{0, 0})
			}
			vs = append(vs, n.Vertices[int(f.Vertex[0])])
			vs = append(vs, n.Vertices[int(f.Vertex[1])])
			vs = append(vs, n.Vertices[int(f.Vertex[2])])
			f.Vertex = [3]uint32{idx, uint32(idx + 1), uint32(idx + 2)}
			idx += 3
		}
	}
	n.Vertices = vs
	n.Normals = vns
	n.TexCoords = vts
}

func (n *MeshNode) ReComputeNormal() {
	normals := make([]vec3.T, len(n.Vertices))
	for _, g := range n.FaceGroup {
		for _, f := range g.Faces {
			pt1 := n.Vertices[f.Vertex[0]]
			pt2 := n.Vertices[f.Vertex[1]]
			pt3 := n.Vertices[f.Vertex[2]]

			sub1 := vec3.Sub(&pt3, &pt2)
			sub2 := vec3.Sub(&pt1, &pt2)

			cro := vec3.Cross(&sub1, &sub2)
			l := cro.Length()
			if l == 0 {
				continue
			}
			weightedNormal := cro.Scale(1 / l)

			normals[f.Vertex[0]].Add(weightedNormal)
			normals[f.Vertex[1]].Add(weightedNormal)
			normals[f.Vertex[2]].Add(weightedNormal)
		}
	}

	for i := range normals {
		normals[i].Normalize()
	}

	n.Normals = normals
}

type InstanceMesh struct {
	Transfors []*dmat.T
	Features  []uint64
	BBox      *[6]float64
	Mesh      *BaseMesh
	Props     []*Properties `json:"props,omitempty"`
	Hash      uint64
}

func (nd *MeshNode) GetBoundbox() *[6]float64 {
	minX := math.MaxFloat64
	minY := math.MaxFloat64
	minZ := math.MaxFloat64
	maxX := -math.MaxFloat64
	maxY := -math.MaxFloat64
	maxZ := -math.MaxFloat64
	for i := range nd.Vertices {
		minX = math.Min(minX, float64(nd.Vertices[i][0]))
		minY = math.Min(minY, float64(nd.Vertices[i][1]))
		minZ = math.Min(minZ, float64(nd.Vertices[i][2]))

		maxX = math.Max(maxX, float64(nd.Vertices[i][0]))
		maxY = math.Max(maxY, float64(nd.Vertices[i][1]))
		maxZ = math.Max(maxZ, float64(nd.Vertices[i][2]))
	}
	return &[6]float64{minX, minY, minZ, maxX, maxY, maxZ}
}

type BaseMesh struct {
	Materials []MeshMaterial `json:"materials,omitempty"`
	Nodes     []*MeshNode    `json:"nodes,omitempty"`
	Code      uint32         `json:"code,omitempty"`
}

type Mesh struct {
	BaseMesh
	Version      uint32 `json:"version"`
	InstanceNode []*InstanceMesh
	Props        *Properties `json:"props,omitempty"`
}

func NewMesh() *Mesh {
	return &Mesh{Version: V5, Props: &Properties{}}
}

func (m *Mesh) NodeCount() int {
	return len(m.Nodes)
}

func (m *Mesh) MaterialCount() int {
	return len(m.Materials)
}

func (m *Mesh) ComputeBBox() dvec3.Box {
	if len(m.Nodes) == 0 {
		return dvec3.Box{}
	}

	bbox := dvec3.MinBox
	for _, nd := range m.Nodes {
		bx := nd.GetBoundbox()
		min := dvec3.T{bx[0], bx[1], bx[2]}
		max := dvec3.T{bx[3], bx[4], bx[5]}
		bbx := dvec3.Box{Min: min, Max: max}
		bbox.Join(&bbx)
	}
	return bbox
}
