package mst

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"

	dmat "github.com/flywave/go3d/float64/mat4"
	dvec3 "github.com/flywave/go3d/float64/vec3"

	"github.com/flywave/go3d/vec2"
	"github.com/flywave/go3d/vec3"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

const MESH_SIGNATURE string = "fwtm"
const MSTEXT string = ".mst"
const V1 uint32 = 1
const V2 uint32 = 2
const V3 uint32 = 3
const V4 uint32 = 4
const V5 uint32 = 5

const (
	MESH_TRIANGLE_MATERIAL_TYPE_COLOR   = 0
	MESH_TRIANGLE_MATERIAL_TYPE_TEXTURE = 1
	MESH_TRIANGLE_MATERIAL_TYPE_PBR     = 2
	MESH_TRIANGLE_MATERIAL_TYPE_LAMBERT = 3
	MESH_TRIANGLE_MATERIAL_TYPE_PHONG   = 4
)

const (
	PBR_MATERIAL_TYPE_LIT        = 0
	PBR_MATERIAL_TYPE_SUBSURFACE = 1
	PBR_MATERIAL_TYPE_CLOTH      = 2
)

const (
	TEXTURE_PIXEL_TYPE_UBYTE  = 0
	TEXTURE_PIXEL_TYPE_BYTE   = 1
	TEXTURE_PIXEL_TYPE_USHORT = 2
	TEXTURE_PIXEL_TYPE_SHORT  = 3
	TEXTURE_PIXEL_TYPE_UINT   = 4
	TEXTURE_PIXEL_TYPE_INT    = 5
	TEXTURE_PIXEL_TYPE_HALF   = 6
	TEXTURE_PIXEL_TYPE_FLOAT  = 7
)

const (
	TEXTURE_FORMAT_R               = 0
	TEXTURE_FORMAT_R_INTEGER       = 1
	TEXTURE_FORMAT_RG              = 2
	TEXTURE_FORMAT_RG_INTEGER      = 3
	TEXTURE_FORMAT_RGB             = 4
	TEXTURE_FORMAT_RGB_INTEGER     = 5
	TEXTURE_FORMAT_RGBA            = 6
	TEXTURE_FORMAT_RGBA_INTEGER    = 7
	TEXTURE_FORMAT_RGBM            = 8
	TEXTURE_FORMAT_DEPTH_COMPONENT = 9
	TEXTURE_FORMAT_DEPTH_STENCIL   = 10
	TEXTURE_FORMAT_ALPHA           = 11
)

const (
	TEXTURE_COMPRESSED_ZLIB = 1
)

type MeshMaterial interface {
	HasTexture() bool
	GetTexture() *Texture
	GetColor() [3]byte
	GetEmissive() [3]byte
}

type Texture struct {
	Id         int32     `json:"id"`
	Name       string    `json:"name"`
	Size       [2]uint64 `json:"size"`
	Format     uint16    `json:"format"`
	Type       uint16    `json:"type"`
	Compressed uint16    `json:"compressed"`
	Data       []byte    `json:"-"`
	Repeated   bool      `json:"repeated"`
}

type BaseMaterial struct {
	Color        [3]byte `json:"color"`
	Transparency float32 `json:"transparency"`
}

func (m *BaseMaterial) HasTexture() bool {
	return false
}

func (m *BaseMaterial) GetEmissive() [3]byte {
	return [3]byte{0, 0, 0}
}

func (m *BaseMaterial) GetTexture() *Texture {
	return nil
}

func (m *BaseMaterial) GetColor() [3]byte {
	return m.Color
}

type TextureMaterial struct {
	BaseMaterial
	Texture *Texture `json:"texture,omitempty"`
	Normal  *Texture `json:"normal,omitempty"`
}

func (m *TextureMaterial) HasTexture() bool {
	return m.Texture != nil
}

func (m *TextureMaterial) GetTexture() *Texture {
	return m.Texture
}

func (m *TextureMaterial) HasNormalTexture() bool {
	return m.Normal != nil
}

func (m *TextureMaterial) GetNormalTexture() *Texture {
	return m.Normal
}

type PbrMaterial struct {
	TextureMaterial
	Emissive            [3]byte `json:"emissive"`
	Metallic            float32 `json:"metallic"`
	Roughness           float32 `json:"roughness"`
	Reflectance         float32 `json:"reflectance"`
	AmbientOcclusion    float32 `json:"ambientOcclusion"`
	ClearCoat           float32 `json:"clearCoat"`
	ClearCoatRoughness  float32 `json:"clearCoatRoughness"`
	ClearCoatNormal     [3]byte `json:"clearCoatNormal"`
	Anisotropy          float32 `json:"anisotropy"`
	AnisotropyDirection vec3.T  `json:"anisotropyDirection"`
	Thickness           float32 `json:"thickness"`       // subsurface only
	SubSurfacePower     float32 `json:"subSurfacePower"` // subsurface only
	SheenColor          [3]byte `json:"sheenColor"`      // cloth only
	SubSurfaceColor     [3]byte `json:"subSurfaceColor"` // subsurface or cloth
}

func (m *PbrMaterial) GetEmissive() [3]byte {
	return m.Emissive
}

type LambertMaterial struct {
	TextureMaterial
	Ambient  [3]byte `json:"ambient"`
	Diffuse  [3]byte `json:"diffuse"`
	Emissive [3]byte `json:"emissive"`
}

type PhongMaterial struct {
	LambertMaterial
	Specular    [3]byte `json:"specular"`
	Shininess   float32 `json:"shininess"`
	Specularity float32 `json:"specularity"`
}

func (m *LambertMaterial) GetEmissive() [3]byte {
	return m.Emissive
}

type Face struct {
	Vertex [3]uint32
	Normal *[3]uint32
	Uv     *[3]uint32
}
type MeshTriangle struct {
	Batchid int32   `json:"batchid"`
	Faces   []*Face `json:"faces"`
}

type MeshOutline struct {
	Batchid int32       `json:"batchid"`
	Edges   [][2]uint32 `json:"edges"`
}

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
	Props     *Properties `json:"props,omitempty"`
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

func toLittleByteOrder(v interface{}) []byte {
	var buf []byte
	b := bytes.NewBuffer(buf)
	e := binary.Write(b, binary.LittleEndian, v)
	if e != nil {
		return nil
	}
	return b.Bytes()
}

func writeLittleByte(wt io.Writer, v interface{}) {
	buf := toLittleByteOrder(v)
	if buf != nil {
		wt.Write(buf)
	}
}

func readLittleByte(rd io.Reader, v interface{}) error {
	return binary.Read(rd, binary.LittleEndian, v)
}

func BaseMaterialMarshal(wt io.Writer, mtl *BaseMaterial) {
	writeLittleByte(wt, &mtl.Color)
	writeLittleByte(wt, &mtl.Transparency)
}

func BaseMaterialUnMarshal(rd io.Reader) *BaseMaterial {
	mtl := BaseMaterial{}
	readLittleByte(rd, mtl.Color[:])
	readLittleByte(rd, &mtl.Transparency)
	return &mtl
}

func TextureMarshal(wt io.Writer, tex *Texture) {
	writeLittleByte(wt, tex.Id)
	writeLittleByte(wt, uint32(len(tex.Name)))
	wt.Write([]byte(tex.Name))
	writeLittleByte(wt, &tex.Size)
	writeLittleByte(wt, tex.Format)
	writeLittleByte(wt, tex.Type)
	writeLittleByte(wt, tex.Compressed)
	writeLittleByte(wt, uint32(len(tex.Data)))
	wt.Write(tex.Data)
	writeLittleByte(wt, tex.Repeated)
}

func TextureUnMarshal(rd io.Reader) *Texture {
	tex := &Texture{}
	readLittleByte(rd, &tex.Id)
	var name_size uint32
	readLittleByte(rd, &name_size)
	nm := make([]byte, name_size)
	rd.Read(nm)
	tex.Name = string(nm)
	readLittleByte(rd, &tex.Size)
	readLittleByte(rd, &tex.Format)
	readLittleByte(rd, &tex.Type)
	readLittleByte(rd, &tex.Compressed)
	var tex_size uint32
	readLittleByte(rd, &tex_size)
	tex.Data = make([]byte, tex_size)
	readLittleByte(rd, tex.Data)
	readLittleByte(rd, &tex.Repeated)
	return tex
}

func TextureMaterialMarshal(wt io.Writer, mtl *TextureMaterial) {
	BaseMaterialMarshal(wt, &mtl.BaseMaterial)
	if mtl.Texture != nil {
		writeLittleByte(wt, uint16(1))
		TextureMarshal(wt, mtl.Texture)
	} else {
		writeLittleByte(wt, uint16(0))
	}
	if mtl.Normal != nil {
		writeLittleByte(wt, uint16(1))
		TextureMarshal(wt, mtl.Normal)
	} else {
		writeLittleByte(wt, uint16(0))
	}
}

func TextureMaterialUnMarshal(rd io.Reader) *TextureMaterial {
	tmtl := TextureMaterial{}
	bmt := BaseMaterialUnMarshal(rd)
	tmtl.BaseMaterial = *bmt
	var hasTex uint16
	readLittleByte(rd, &hasTex)
	if hasTex == 1 {
		tmtl.Texture = TextureUnMarshal(rd)
	}
	readLittleByte(rd, &hasTex)
	if hasTex == 1 {
		tmtl.Normal = TextureUnMarshal(rd)
	}
	return &tmtl
}

func PbrMaterialMarshal(wt io.Writer, mtl *PbrMaterial, v uint32) {
	TextureMaterialMarshal(wt, &mtl.TextureMaterial)
	writeLittleByte(wt, mtl.Emissive[:])
	if v < V2 {
		writeLittleByte(wt, byte(255))
	}
	writeLittleByte(wt, &mtl.Metallic)
	writeLittleByte(wt, &mtl.Roughness)
	writeLittleByte(wt, &mtl.Reflectance)
	writeLittleByte(wt, &mtl.AmbientOcclusion)
	writeLittleByte(wt, &mtl.ClearCoat)
	writeLittleByte(wt, &mtl.ClearCoatRoughness)
	writeLittleByte(wt, mtl.ClearCoatNormal[:])
	writeLittleByte(wt, &mtl.Anisotropy)
	writeLittleByte(wt, mtl.AnisotropyDirection[:])
	writeLittleByte(wt, &mtl.Thickness)
	writeLittleByte(wt, &mtl.SubSurfacePower)
	writeLittleByte(wt, mtl.SheenColor[:])
	writeLittleByte(wt, mtl.SubSurfaceColor[:])
}

func PbrMaterialUnMarshal(rd io.Reader, v uint32) *PbrMaterial {
	mtl := PbrMaterial{}
	tmtl := TextureMaterialUnMarshal(rd)
	mtl.TextureMaterial = *tmtl
	readLittleByte(rd, mtl.Emissive[:])
	if v < V2 {
		var b byte
		readLittleByte(rd, &b)
	}
	readLittleByte(rd, &mtl.Metallic)
	readLittleByte(rd, &mtl.Roughness)
	readLittleByte(rd, &mtl.Reflectance)
	readLittleByte(rd, &mtl.AmbientOcclusion)
	readLittleByte(rd, &mtl.ClearCoat)
	readLittleByte(rd, &mtl.ClearCoatRoughness)
	readLittleByte(rd, &mtl.ClearCoatNormal)
	readLittleByte(rd, &mtl.Anisotropy)
	readLittleByte(rd, mtl.AnisotropyDirection[:])
	readLittleByte(rd, &mtl.Thickness)
	readLittleByte(rd, &mtl.SubSurfacePower)
	readLittleByte(rd, &mtl.SheenColor)
	readLittleByte(rd, mtl.SubSurfaceColor[:])
	return &mtl
}

func LambertMaterialMarshal(wt io.Writer, mtl *LambertMaterial) {
	TextureMaterialMarshal(wt, &mtl.TextureMaterial)
	writeLittleByte(wt, mtl.Ambient[:])
	writeLittleByte(wt, mtl.Diffuse[:])
	writeLittleByte(wt, mtl.Emissive[:])
}

func LambertMaterialUnMarshal(rd io.Reader) *LambertMaterial {
	mtl := LambertMaterial{}
	tmt := TextureMaterialUnMarshal(rd)
	mtl.TextureMaterial = *tmt
	readLittleByte(rd, mtl.Ambient[:])
	readLittleByte(rd, mtl.Diffuse[:])
	readLittleByte(rd, mtl.Emissive[:])
	return &mtl
}

func PhongMaterialMarshal(wt io.Writer, mtl *PhongMaterial) {
	LambertMaterialMarshal(wt, &mtl.LambertMaterial)
	writeLittleByte(wt, mtl.Specular[:])
	writeLittleByte(wt, &mtl.Shininess)
	writeLittleByte(wt, &mtl.Specularity)
}

func PhongMaterialUnMarshal(rd io.Reader) *PhongMaterial {
	mtl := PhongMaterial{}
	mt := LambertMaterialUnMarshal(rd)
	mtl.LambertMaterial = *mt
	readLittleByte(rd, mtl.Specular[:])
	readLittleByte(rd, &mtl.Shininess)
	readLittleByte(rd, &mtl.Specularity)
	return &mtl
}

func MaterialMarshal(wt io.Writer, mt MeshMaterial, v uint32) {
	switch mtl := mt.(type) {
	case *BaseMaterial:
		writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_COLOR))
		BaseMaterialMarshal(wt, mtl)
	case *TextureMaterial:
		writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_TEXTURE))
		TextureMaterialMarshal(wt, mtl)
	case *PbrMaterial:
		writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_PBR))
		PbrMaterialMarshal(wt, mtl, v)
	case *LambertMaterial:
		writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_LAMBERT))
		LambertMaterialMarshal(wt, mtl)
	case *PhongMaterial:
		writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_PHONG))
		PhongMaterialMarshal(wt, mtl)
	}
}

func MaterialUnMarshal(rd io.Reader, v uint32) MeshMaterial {
	var ty uint32
	readLittleByte(rd, &ty)
	switch int(ty) {
	case MESH_TRIANGLE_MATERIAL_TYPE_COLOR:
		return BaseMaterialUnMarshal(rd)
	case MESH_TRIANGLE_MATERIAL_TYPE_TEXTURE:
		return TextureMaterialUnMarshal(rd)
	case MESH_TRIANGLE_MATERIAL_TYPE_PBR:
		return PbrMaterialUnMarshal(rd, v)
	case MESH_TRIANGLE_MATERIAL_TYPE_LAMBERT:
		return LambertMaterialUnMarshal(rd)
	case MESH_TRIANGLE_MATERIAL_TYPE_PHONG:
		return PhongMaterialUnMarshal(rd)
	default:
		return nil
	}
}

func MtlsMarshal(wt io.Writer, mtls []MeshMaterial, v uint32) {
	writeLittleByte(wt, uint32(len(mtls)))
	for _, mtl := range mtls {
		MaterialMarshal(wt, mtl, v)
	}
}

func MtlsUnMarshal(rd io.Reader, v uint32) []MeshMaterial {
	var size uint32
	readLittleByte(rd, &size)
	mtls := make([]MeshMaterial, size)
	for i := 0; i < int(size); i++ {
		mtls[i] = MaterialUnMarshal(rd, v)
	}
	return mtls
}

func MeshTriangleMarshal(wt io.Writer, nd *MeshTriangle) {
	writeLittleByte(wt, nd.Batchid)
	writeLittleByte(wt, uint32(len(nd.Faces)))
	for _, f := range nd.Faces {
		writeLittleByte(wt, &f.Vertex)
	}
}

func MeshTriangleUnMarshal(rd io.Reader) *MeshTriangle {
	nd := MeshTriangle{}
	readLittleByte(rd, &nd.Batchid)
	var size uint32
	readLittleByte(rd, &size)
	nd.Faces = make([]*Face, size)
	for i := 0; i < int(size); i++ {
		f := &Face{}
		nd.Faces[i] = f
		readLittleByte(rd, &f.Vertex)
	}
	return &nd
}

// 辅助函数：写入小端格式的uint32
func writeLittleUint32(w io.Writer, v uint32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

// 辅助函数：写入小端格式的int64
func writeLittleInt64(w io.Writer, v int64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(v))
	_, err := w.Write(buf[:])
	return err
}

// 辅助函数：写入小端格式的float64
func writeLittleFloat64(w io.Writer, v float64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(v))
	_, err := w.Write(buf[:])
	return err
}

// 辅助函数：写入小端格式的uint8
func writeLittleUint8(w io.Writer, v uint8) error {
	_, err := w.Write([]byte{v})
	return err
}

func MeshOutlineMarshal(wt io.Writer, nd *MeshOutline) {
	writeLittleByte(wt, nd.Batchid)
	writeLittleByte(wt, uint32(len(nd.Edges)))
	for _, e := range nd.Edges {
		writeLittleByte(wt, &e)
	}
}

func MeshOutlineUnMarshal(rd io.Reader) *MeshOutline {
	nd := MeshOutline{}
	readLittleByte(rd, &nd.Batchid)
	var size uint32
	readLittleByte(rd, &size)
	nd.Edges = make([][2]uint32, size)
	for i := 0; i < int(size); i++ {
		readLittleByte(rd, &nd.Edges[i])
	}
	return &nd
}

func MeshNodeMarshal(wt io.Writer, nd *MeshNode) {
	writeLittleByte(wt, uint32(len(nd.Vertices)))
	for i := range nd.Vertices {
		writeLittleByte(wt, nd.Vertices[i][:])
	}
	writeLittleByte(wt, uint32(len(nd.Normals)))
	for i := range nd.Normals {
		writeLittleByte(wt, nd.Normals[i][:])
	}
	writeLittleByte(wt, uint32(len(nd.Colors)))
	for i := range nd.Colors {
		writeLittleByte(wt, nd.Colors[i][:])

	}
	writeLittleByte(wt, uint32(len(nd.TexCoords)))
	for i := range nd.TexCoords {
		writeLittleByte(wt, nd.TexCoords[i][:])
	}
	if nd.Mat != nil {
		writeLittleByte(wt, uint8(1))
		writeLittleByte(wt, nd.Mat[0][:])
		writeLittleByte(wt, nd.Mat[1][:])
		writeLittleByte(wt, nd.Mat[2][:])
		writeLittleByte(wt, nd.Mat[3][:])
	} else {
		writeLittleByte(wt, uint8(0))
	}

	writeLittleByte(wt, uint32(len(nd.FaceGroup)))
	for _, fg := range nd.FaceGroup {
		MeshTriangleMarshal(wt, fg)
	}

	writeLittleByte(wt, uint32(len(nd.EdgeGroup)))
	for _, eg := range nd.EdgeGroup {
		MeshOutlineMarshal(wt, eg)
	}
	// V5 版本序列化新增属性
	if nd.Props != nil && len(*nd.Props) > 0 {
		PropertiesMarshal(wt, nd.Props)
	} else {
		// 如果Props为nil，写入size为0
		writeLittleByte(wt, uint32(0))
	}
}

func MeshNodeUnMarshal(rd io.Reader) *MeshNode {
	nd := MeshNode{}
	var size uint32
	readLittleByte(rd, &size)
	nd.Vertices = make([]vec3.T, size)
	for i := range nd.Vertices {
		readLittleByte(rd, nd.Vertices[i][:])
	}
	readLittleByte(rd, &size)
	nd.Normals = make([]vec3.T, size)
	for i := range nd.Normals {
		readLittleByte(rd, nd.Normals[i][:])
	}
	readLittleByte(rd, &size)
	nd.Colors = make([][3]byte, size)
	for i := range nd.Colors {
		readLittleByte(rd, nd.Colors[i][:])
	}

	readLittleByte(rd, &size)
	nd.TexCoords = make([]vec2.T, size)
	for i := range nd.TexCoords {
		readLittleByte(rd, &nd.TexCoords[i])
	}
	var isMat uint8
	readLittleByte(rd, &isMat)
	if isMat == 1 {
		nd.Mat = &dmat.T{}
		readLittleByte(rd, nd.Mat[0][:])
		readLittleByte(rd, nd.Mat[1][:])
		readLittleByte(rd, nd.Mat[2][:])
		readLittleByte(rd, nd.Mat[3][:])
	}

	readLittleByte(rd, &size)
	nd.FaceGroup = make([]*MeshTriangle, size)
	for i := 0; i < int(size); i++ {
		nd.FaceGroup[i] = MeshTriangleUnMarshal(rd)
	}

	readLittleByte(rd, &size)
	nd.EdgeGroup = make([]*MeshOutline, size)
	for i := 0; i < int(size); i++ {
		nd.EdgeGroup[i] = MeshOutlineUnMarshal(rd)
	}
	// V5 版本反序列化新增属性
	nd.Props = PropertiesUnMarshal(rd)
	return &nd
}

func MeshNodesMarshal(wt io.Writer, nds []*MeshNode) {
	writeLittleByte(wt, uint32(len(nds)))
	for _, nd := range nds {
		MeshNodeMarshal(wt, nd)
	}
}

func MeshNodesMarshalWithVersion(wt io.Writer, nds []*MeshNode, v uint32) {
	writeLittleByte(wt, uint32(len(nds)))
	for _, nd := range nds {
		if v >= V5 {
			MeshNodeMarshal(wt, nd)
		} else {
			MeshNodeMarshalWithoutProps(wt, nd)
		}
	}
}

func MeshNodesUnMarshal(rd io.Reader) []*MeshNode {
	var size uint32
	readLittleByte(rd, &size)
	nds := make([]*MeshNode, size)
	for i := range nds {
		nds[i] = MeshNodeUnMarshal(rd)
	}
	return nds
}

func MeshMarshal(wt io.Writer, ms *Mesh) {
	wt.Write([]byte(MESH_SIGNATURE))
	writeLittleByte(wt, ms.Version)
	// V4及以上版本序列化Code字段
	if ms.Version >= V4 {
		writeLittleByte(wt, ms.BaseMesh.Code)
	}
	MtlsMarshal(wt, ms.Materials, ms.Version)
	MeshNodesMarshalWithVersion(wt, ms.Nodes, ms.Version)
	MeshInstanceNodesMarshal(wt, ms.InstanceNode, ms.Version)
	// V5 版本序列化新增属性
	if ms.Version >= V5 {
		if ms.Props != nil && len(*ms.Props) > 0 {
			// 先写入标记位1表示有Properties
			writeLittleByte(wt, uint32(1))
			if err := PropertiesMarshal(wt, ms.Props); err != nil {
				return
			}
		} else {
			// 如果Props为nil，写入标记位0
			writeLittleByte(wt, uint32(0))
		}
	}
}

func MeshUnMarshal(rd io.Reader) *Mesh {
	ms := Mesh{}
	sig := make([]byte, 4)
	rd.Read(sig)
	readLittleByte(rd, &ms.Version)
	// V4及以上版本反序列化Code字段
	if ms.Version >= V4 {
		var code uint32
		readLittleByte(rd, &code)
		ms.BaseMesh.Code = code
	}
	ms.Materials = MtlsUnMarshal(rd, ms.Version)
	// 对于Mesh中的Mesh.Nodes，我们应该使用带版本的函数来正确处理Props字段
	if ms.Version >= V5 {
		ms.Nodes = MeshNodesUnMarshalWithVersion(rd, ms.Version)
	} else {
		ms.Nodes = MeshNodesUnMarshal(rd)
	}
	ms.InstanceNode = MeshInstanceNodesUnMarshal(rd, ms.Version)
	// V5 版本反序列化新增属性
	if ms.Version >= V5 {
		var hasProps uint32
		if err := readLittleByte(rd, &hasProps); err != nil {
			return nil
		}
		if hasProps > 0 {
			ms.Props = PropertiesUnMarshal(rd)
			if ms.Props == nil {
				return nil
			}
		} else {
			ms.Props = nil
		}
	}
	return &ms
}

func MeshInstanceNodesMarshal(wt io.Writer, instNd []*InstanceMesh, v uint32) {
	writeLittleByte(wt, uint32(len(instNd)))
	for _, nd := range instNd {
		MeshInstanceNodeMarshal(wt, nd, v)
	}
}

// MeshNodesMarshalForInstanceMesh 序列化InstanceMesh中的MeshNode，不序列化Props属性
func MeshNodesMarshalForInstanceMesh(wt io.Writer, nds []*MeshNode) {
	writeLittleByte(wt, uint32(len(nds)))
	for _, nd := range nds {
		MeshNodeMarshalWithoutProps(wt, nd)
	}
}

// MeshNodeMarshalWithoutProps 序列化MeshNode，不序列化Props属性
func MeshNodeMarshalWithoutProps(wt io.Writer, nd *MeshNode) {
	writeLittleByte(wt, uint32(len(nd.Vertices)))
	for i := range nd.Vertices {
		writeLittleByte(wt, nd.Vertices[i][:])
	}
	writeLittleByte(wt, uint32(len(nd.Normals)))
	for i := range nd.Normals {
		writeLittleByte(wt, nd.Normals[i][:])
	}
	writeLittleByte(wt, uint32(len(nd.Colors)))
	for i := range nd.Colors {
		writeLittleByte(wt, nd.Colors[i][:])

	}
	writeLittleByte(wt, uint32(len(nd.TexCoords)))
	for i := range nd.TexCoords {
		writeLittleByte(wt, nd.TexCoords[i][:])
	}
	if nd.Mat != nil {
		writeLittleByte(wt, uint8(1))
		writeLittleByte(wt, nd.Mat[0][:])
		writeLittleByte(wt, nd.Mat[1][:])
		writeLittleByte(wt, nd.Mat[2][:])
		writeLittleByte(wt, nd.Mat[3][:])
	} else {
		writeLittleByte(wt, uint8(0))
	}

	writeLittleByte(wt, uint32(len(nd.FaceGroup)))
	for _, fg := range nd.FaceGroup {
		MeshTriangleMarshal(wt, fg)
	}

	writeLittleByte(wt, uint32(len(nd.EdgeGroup)))
	for _, eg := range nd.EdgeGroup {
		MeshOutlineMarshal(wt, eg)
	}
}

func MeshInstanceNodeMarshal(wt io.Writer, instNd *InstanceMesh, v uint32) {
	writeLittleByte(wt, uint32(len(instNd.Transfors)))
	for _, mt := range instNd.Transfors {
		writeLittleByte(wt, mt[0][:])
		writeLittleByte(wt, mt[1][:])
		writeLittleByte(wt, mt[2][:])
		writeLittleByte(wt, mt[3][:])
	}
	writeLittleByte(wt, uint32(len(instNd.Features)))
	for _, f := range instNd.Features {
		writeLittleByte(wt, f)
	}
	writeLittleByte(wt, instNd.BBox)
	// 序列化Mesh字段
	MtlsMarshal(wt, instNd.Mesh.Materials, v)
	// 修复：使用正确的函数来序列化Mesh.Nodes，确保Props字段能被正确处理
	// 对于InstanceMesh中的Mesh.Nodes，我们不应该序列化Props属性，因为Props是InstanceMesh的独立属性
	MeshNodesMarshalForInstanceMesh(wt, instNd.Mesh.Nodes)
	// V4及以上版本序列化Code字段
	if v >= V4 {
		writeLittleByte(wt, instNd.Mesh.Code)
	}
	// V5 版本序列化新增属性
	if v >= V5 {
		var hasProps uint32 = 0
		if instNd.Props != nil && len(*instNd.Props) > 0 {
			hasProps = 1
		}
		// 统一写入hasProps标记
		if err := writeLittleUint32(wt, hasProps); err != nil {
			return
		}
		if hasProps == 1 {
			if err := PropertiesMarshal(wt, instNd.Props); err != nil {
				return
			}
		}
	}
	writeLittleByte(wt, instNd.Hash)
}

// MeshNodesUnMarshalForInstanceMesh 反序列化InstanceMesh中的MeshNode，不读取Props属性
func MeshNodesUnMarshalForInstanceMesh(rd io.Reader) []*MeshNode {
	var size uint32
	readLittleByte(rd, &size)
	nds := make([]*MeshNode, size)
	for i := range nds {
		nds[i] = MeshNodeUnMarshalWithoutProps(rd)
	}
	return nds
}

func MeshInstanceNodeUnMarshal(rd io.Reader, v uint32) *InstanceMesh {
	inst := &InstanceMesh{}
	var size uint32
	readLittleByte(rd, &size)
	inst.Transfors = make([]*dmat.T, size)
	for i := range inst.Transfors {
		mt := &dmat.T{}
		readLittleByte(rd, &mt[0])
		readLittleByte(rd, &mt[1])
		readLittleByte(rd, &mt[2])
		readLittleByte(rd, &mt[3])
		inst.Transfors[i] = mt
	}
	var fsize uint32
	readLittleByte(rd, &fsize)
	inst.Features = make([]uint64, fsize)
	if v < V3 {
		fs := make([]uint32, fsize)
		readLittleByte(rd, &fs)
		for i, f := range fs {
			inst.Features[i] = uint64(f)
		}
	} else {
		readLittleByte(rd, &inst.Features)
	}

	inst.BBox = &[6]float64{}
	readLittleByte(rd, inst.BBox)
	// 反序列化Mesh字段
	inst.Mesh = &BaseMesh{}
	inst.Mesh.Materials = MtlsUnMarshal(rd, v)
	// 修复：使用正确的函数来反序列化Mesh.Nodes，确保Props字段能被正确处理
	// 对于InstanceMesh中的Mesh.Nodes，我们不应该读取Props属性，因为Props是InstanceMesh的独立属性
	inst.Mesh.Nodes = MeshNodesUnMarshalForInstanceMesh(rd)
	// V4及以上版本反序列化Code字段
	if v >= V4 {
		readLittleByte(rd, &inst.Mesh.Code)
	}
	// V5 版本反序列化新增属性
	if v >= V5 {
		var hasProps uint32
		if err := readLittleByte(rd, &hasProps); err != nil {
			return nil
		}
		if hasProps > 0 {
			inst.Props = PropertiesUnMarshal(rd)
			if inst.Props == nil {
				return nil
			}
		} else {
			inst.Props = nil
		}
	} else {
		// For versions less than V5, ensure Props is nil
		inst.Props = nil
	}
	readLittleByte(rd, &inst.Hash)
	return inst
}

func MeshInstanceNodesUnMarshal(rd io.Reader, v uint32) []*InstanceMesh {
	var size uint32
	readLittleByte(rd, &size)
	nds := make([]*InstanceMesh, size)
	for i := range nds {
		nds[i] = MeshInstanceNodeUnMarshal(rd, v)
	}
	return nds
}

func MeshReadFrom(path string) (*Mesh, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return MeshUnMarshal(f), nil
}

func MeshWriteTo(path string, ms *Mesh) error {
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	MeshMarshal(f, ms)
	return nil
}

func CompressImage(buf []byte) []byte {
	var bt []byte
	bf := bytes.NewBuffer(bt)
	w := zlib.NewWriter(bf)
	w.Write(buf)
	w.Close()
	return bf.Bytes()
}

func DecompressImage(src []byte) ([]byte, error) {
	bf := bytes.NewBuffer(src)
	r, er := zlib.NewReader(bf)
	if er != nil {
		return nil, er
	}
	return io.ReadAll(r)
}

func LoadTexture(tex *Texture, flipY bool) (image.Image, error) {
	w := int(tex.Size[0])
	h := int(tex.Size[1])
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	data := tex.Data
	var sz int
	switch tex.Format {
	case TEXTURE_FORMAT_RGB:
		sz = 3
	case TEXTURE_FORMAT_RGBA:
		sz = 4
	case TEXTURE_FORMAT_R:
		sz = 1
	}
	var e error
	if tex.Compressed == TEXTURE_COMPRESSED_ZLIB {
		data, e = DecompressImage(data)
		if e != nil && e.Error() != "EOF" {
			return nil, e
		}
	}

	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			p := i*w*sz + j*sz
			var c color.NRGBA
			switch sz {
			case 4:
				c = color.NRGBA{R: data[p], G: data[p+1], B: data[p+2], A: data[p+3]}
			case 3:
				c = color.NRGBA{R: data[p], G: data[p+1], B: data[p+2], A: 255}
			case 1:
				c = color.NRGBA{R: data[p], G: data[p], B: data[p], A: 255}
			}

			y := i
			if flipY {
				y = h - i - 1
			}
			img.Set(j, y, c)
		}
	}
	return img, nil
}

func CreateTexture(name string, repet bool) (*Texture, error) {
	reader, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	_, format, err := image.DecodeConfig(reader)
	if err != nil {
		return nil, err
	}
	reader.Seek(0, io.SeekStart)
	var img image.Image
	switch format {
	case "jpeg", "jpg":
		img, err = jpeg.Decode(reader)
	case "png":
		img, err = png.Decode(reader)
	case "gif":
		img, err = gif.Decode(reader)
	case "bmp":
		img, err = bmp.Decode(reader)
	case "tif", "tiff":
		img, err = tiff.Decode(reader)
	default:
		return nil, errors.New("unknow format")
	}
	if err != nil {
		return nil, err
	}
	return CreateTextureFromImage(img, name, repet)
}

func CreateTextureFromImage(img image.Image, name string, repet bool) (*Texture, error) {
	bd := img.Bounds()
	buf1 := []byte{}

	for y := 0; y < bd.Dy(); y++ {
		for x := 0; x < bd.Dx(); x++ {
			cl := img.At(x, y)
			r, g, b, a := color.RGBAModel.Convert(cl).RGBA()
			buf1 = append(buf1, byte(r&0xff), byte(g&0xff), byte(b&0xff), byte(a&0xff))
		}
	}
	t := &Texture{}
	_, fn := filepath.Split(name)
	t.Name = fn
	t.Format = TEXTURE_FORMAT_RGBA
	t.Size = [2]uint64{uint64(bd.Dx()), uint64(bd.Dy())}
	t.Compressed = TEXTURE_COMPRESSED_ZLIB
	t.Data = CompressImage(buf1)
	t.Repeated = repet
	return t, nil
}

type PropsType int

const (
	PROP_TYPE_STRING = iota
	PROP_TYPE_INT
	PROP_TYPE_FLOAT
	PROP_TYPE_BOOL
	PROP_TYPE_ARRAY
	PROP_TYPE_MAP
)

type PropsValue struct {
	Type  PropsType
	Value interface{}
}

type Properties map[string]PropsValue

// PropertiesMarshal 序列化Properties
func PropertiesMarshal(wt io.Writer, props *Properties) error {
	// 嵌套函数：序列化单个PropsValue
	marshalPropsValue := func(wt io.Writer, value PropsValue) error {
		switch value.Type {
		case PROP_TYPE_STRING:
			str := value.Value.(string)
			if err := writeLittleUint32(wt, uint32(len(str))); err != nil {
				return fmt.Errorf("write string len failed: %w", err)
			}
			if _, err := wt.Write([]byte(str)); err != nil {
				return fmt.Errorf("write string content failed: %w", err)
			}
		case PROP_TYPE_INT:
			intVal := value.Value.(int64)
			if err := writeLittleInt64(wt, intVal); err != nil {
				return fmt.Errorf("write int64 failed: %w", err)
			}
		case PROP_TYPE_FLOAT:
			floatVal := value.Value.(float64)
			if err := writeLittleFloat64(wt, floatVal); err != nil {
				return fmt.Errorf("write float64 failed: %w", err)
			}
		case PROP_TYPE_BOOL:
			val := uint8(0)
			if value.Value.(bool) {
				val = 1
			}
			if err := writeLittleUint8(wt, val); err != nil {
				return fmt.Errorf("write bool failed: %w", err)
			}
		case PROP_TYPE_ARRAY:
			arr := value.Value.([]PropsValue)
			if err := writeLittleUint32(wt, uint32(len(arr))); err != nil {
				return fmt.Errorf("write array len failed: %w", err)
			}
			for _, item := range arr {
				if err := writeLittleUint32(wt, uint32(item.Type)); err != nil {
					return fmt.Errorf("write array item type failed: %w", err)
				}
				if err := marshalPropsValue(wt, item); err != nil {
					return fmt.Errorf("write array item failed: %w", err)
				}
			}
		case PROP_TYPE_MAP:
			subProps := value.Value.(Properties)
			if err := PropertiesMarshal(wt, &subProps); err != nil {
				return fmt.Errorf("write map properties failed: %w", err)
			}
		}
		return nil
	}

	if props == nil {
		if err := writeLittleUint32(wt, 0); err != nil {
			return fmt.Errorf("write nil marker failed: %w", err)
		}
		return nil
	}

	// 写入Properties数量
	propsCount := uint32(len(*props))
	if err := writeLittleUint32(wt, propsCount); err != nil {
		return fmt.Errorf("write properties count failed: %w", err)
	}

	for key, value := range *props {
		// 写入key长度
		keyLen := uint32(len(key))
		if err := writeLittleUint32(wt, keyLen); err != nil {
			return fmt.Errorf("write key len failed: %w", err)
		}
		// 写入key内容
		if _, err := wt.Write([]byte(key)); err != nil {
			return fmt.Errorf("write key content failed: %w", err)
		}

		// 写入类型
		if err := writeLittleUint32(wt, uint32(value.Type)); err != nil {
			return fmt.Errorf("write value type failed: %w", err)
		}

		// 根据类型写入值
		if err := marshalPropsValue(wt, value); err != nil {
			return fmt.Errorf("write value failed: %w", err)
		}
	}
	return nil
}

// PropertiesUnMarshal 反序列化Properties
func PropertiesUnMarshal(rd io.Reader) *Properties {
	// 读取Properties数量
	var size uint32
	if err := readLittleByte(rd, &size); err != nil {
		return nil
	}

	// 安全检查
	if size > 1000 { // 设置合理的上限
		return nil
	}

	props := make(Properties)
	for i := uint32(0); i < size; i++ {
		// 读取key长度
		var keyLen uint32
		if err := readLittleByte(rd, &keyLen); err != nil {
			return nil
		}

		// 安全检查
		if keyLen > 100 { // 设置合理的key长度上限
			return nil
		}

		// 读取key内容
		keyBytes := make([]byte, keyLen)
		if _, err := io.ReadFull(rd, keyBytes); err != nil {
			return nil
		}
		key := string(keyBytes)

		// 读取类型
		var propType uint32
		if err := readLittleByte(rd, &propType); err != nil {
			return nil
		}

		// 根据类型读取值
		value := unmarshalPropsValue(rd, PropsType(propType))
		if value.Type == -1 { // 表示反序列化失败
			return nil
		}

		// 类型验证
		if uint32(value.Type) != propType {
			return nil
		}

		props[key] = value
	}

	return &props
}

// 辅助函数，用于反序列化单个PropsValue
func unmarshalPropsValue(rd io.Reader, propType PropsType) PropsValue {
	var value interface{}
	var err error

	switch propType {
	case PROP_TYPE_STRING:
		var strLen uint32
		if err = readLittleByte(rd, &strLen); err != nil {
			return PropsValue{Type: -1}
		}
		// 添加安全检查
		if strLen > 100000 {
			return PropsValue{Type: -1}
		}
		strBytes := make([]byte, strLen)
		if _, err = io.ReadFull(rd, strBytes); err != nil {
			return PropsValue{Type: -1}
		}
		value = string(strBytes)
	case PROP_TYPE_INT:
		var intVal int64
		if err = readLittleByte(rd, &intVal); err != nil {
			return PropsValue{Type: -1}
		}
		value = intVal
	case PROP_TYPE_FLOAT:
		var floatVal float64
		if err = readLittleByte(rd, &floatVal); err != nil {
			return PropsValue{Type: -1}
		}
		value = floatVal
	case PROP_TYPE_BOOL:
		var boolVal uint8
		if err = readLittleByte(rd, &boolVal); err != nil {
			return PropsValue{Type: -1}
		}
		value = boolVal == 1
	case PROP_TYPE_ARRAY:
		var arrLen uint32
		if err = readLittleByte(rd, &arrLen); err != nil {
			return PropsValue{Type: -1}
		}
		// 添加安全检查
		if arrLen > 100000 {
			return PropsValue{Type: -1}
		}
		arr := make([]PropsValue, arrLen)
		for i := uint32(0); i < arrLen; i++ {
			var itemType uint32
			if err = readLittleByte(rd, &itemType); err != nil {
				return PropsValue{Type: -1}
			}
			arr[i] = unmarshalPropsValue(rd, PropsType(itemType))
			if arr[i].Type == -1 {
				return PropsValue{Type: -1}
			}
		}
		value = arr
	case PROP_TYPE_MAP:
		subProps := PropertiesUnMarshal(rd)
		if subProps != nil {
			value = *subProps
		} else {
			value = make(Properties)
		}
	default:
		return PropsValue{Type: -1}
	}

	return PropsValue{
		Type:  propType,
		Value: value,
	}
}

// 辅助函数，用于序列化单个PropsValue
func marshalPropsValue(wt io.Writer, value PropsValue) error {
	switch value.Type {
	case PROP_TYPE_STRING:
		str := value.Value.(string)
		writeLittleByte(wt, uint32(len(str)))
		wt.Write([]byte(str))
	case PROP_TYPE_INT:
		writeLittleByte(wt, value.Value.(int64))
	case PROP_TYPE_FLOAT:
		writeLittleByte(wt, value.Value.(float64))
	case PROP_TYPE_BOOL:
		if value.Value.(bool) {
			writeLittleByte(wt, uint8(1))
		} else {
			writeLittleByte(wt, uint8(0))
		}
	case PROP_TYPE_ARRAY:
		arr := value.Value.([]PropsValue)
		writeLittleByte(wt, uint32(len(arr)))
		for _, item := range arr {
			writeLittleByte(wt, uint32(item.Type))
			marshalPropsValue(wt, item)
		}
	case PROP_TYPE_MAP:
		subProps := value.Value.(Properties)
		PropertiesMarshal(wt, &subProps)
	}
	return nil
}

func MeshNodesUnMarshalWithoutProps(rd io.Reader) []*MeshNode {
	var size uint32
	readLittleByte(rd, &size)
	nds := make([]*MeshNode, size)
	for i := range nds {
		nds[i] = MeshNodeUnMarshalWithoutProps(rd)
	}
	return nds
}

func MeshNodeUnMarshalWithoutProps(rd io.Reader) *MeshNode {
	nd := MeshNode{}
	var size uint32
	readLittleByte(rd, &size)
	nd.Vertices = make([]vec3.T, size)
	for i := range nd.Vertices {
		readLittleByte(rd, nd.Vertices[i][:])
	}
	readLittleByte(rd, &size)
	nd.Normals = make([]vec3.T, size)
	for i := range nd.Normals {
		readLittleByte(rd, nd.Normals[i][:])
	}
	// 修复：重新读取颜色数组大小
	var colorSize uint32
	readLittleByte(rd, &colorSize)
	nd.Colors = make([][3]byte, colorSize)
	for i := range nd.Colors {
		readLittleByte(rd, nd.Colors[i][:])
	}

	// 修复：重新读取纹理坐标数组大小
	var texCoordSize uint32
	readLittleByte(rd, &texCoordSize)
	nd.TexCoords = make([]vec2.T, texCoordSize)
	for i := range nd.TexCoords {
		readLittleByte(rd, &nd.TexCoords[i])
	}
	var isMat uint8
	readLittleByte(rd, &isMat)
	if isMat == 1 {
		nd.Mat = &dmat.T{}
		readLittleByte(rd, nd.Mat[0][:])
		readLittleByte(rd, nd.Mat[1][:])
		readLittleByte(rd, nd.Mat[2][:])
		readLittleByte(rd, nd.Mat[3][:])
	}

	readLittleByte(rd, &size)
	nd.FaceGroup = make([]*MeshTriangle, size)
	for i := 0; i < int(size); i++ {
		nd.FaceGroup[i] = MeshTriangleUnMarshal(rd)
	}

	readLittleByte(rd, &size)
	nd.EdgeGroup = make([]*MeshOutline, size)
	for i := 0; i < int(size); i++ {
		nd.EdgeGroup[i] = MeshOutlineUnMarshal(rd)
	}

	nd.Props = nil
	return &nd
}

func MeshNodesUnMarshalWithVersion(rd io.Reader, v uint32) []*MeshNode {
	var size uint32
	readLittleByte(rd, &size)
	nds := make([]*MeshNode, size)
	for i := range nds {
		nds[i] = MeshNodeUnMarshalWithVersion(rd, v)
	}
	return nds
}

func MeshNodeUnMarshalWithVersion(rd io.Reader, v uint32) *MeshNode {
	nd := MeshNode{}
	var size uint32
	readLittleByte(rd, &size)
	nd.Vertices = make([]vec3.T, size)
	for i := range nd.Vertices {
		readLittleByte(rd, nd.Vertices[i][:])
	}
	readLittleByte(rd, &size)
	nd.Normals = make([]vec3.T, size)
	for i := range nd.Normals {
		readLittleByte(rd, nd.Normals[i][:])
	}
	// 修复：重新读取颜色数组大小
	var colorSize uint32
	readLittleByte(rd, &colorSize)
	nd.Colors = make([][3]byte, colorSize)
	for i := range nd.Colors {
		readLittleByte(rd, nd.Colors[i][:])
	}

	// 修复：重新读取纹理坐标数组大小
	var texCoordSize uint32
	readLittleByte(rd, &texCoordSize)
	nd.TexCoords = make([]vec2.T, texCoordSize)
	for i := range nd.TexCoords {
		readLittleByte(rd, &nd.TexCoords[i])
	}
	var isMat uint8
	readLittleByte(rd, &isMat)
	if isMat == 1 {
		nd.Mat = &dmat.T{}
		readLittleByte(rd, nd.Mat[0][:])
		readLittleByte(rd, nd.Mat[1][:])
		readLittleByte(rd, nd.Mat[2][:])
		readLittleByte(rd, nd.Mat[3][:])
	}

	readLittleByte(rd, &size)
	nd.FaceGroup = make([]*MeshTriangle, size)
	for i := 0; i < int(size); i++ {
		nd.FaceGroup[i] = MeshTriangleUnMarshal(rd)
	}

	readLittleByte(rd, &size)
	nd.EdgeGroup = make([]*MeshOutline, size)
	for i := 0; i < int(size); i++ {
		nd.EdgeGroup[i] = MeshOutlineUnMarshal(rd)
	}

	// V5 版本反序列化新增属性
	if v >= V5 {
		nd.Props = PropertiesUnMarshal(rd)
	} else {
		nd.Props = nil
	}
	return &nd
}
