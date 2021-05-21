package mst

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"image"
	"image/color"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	dmat "github.com/flywave/go3d/float64/mat4"
	"github.com/flywave/go3d/vec2"
	"github.com/flywave/go3d/vec3"
)

const MESH_SIGNATURE string = "fwtm"
const MSTEXT string = ".mst"

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
}

type Texture struct {
	Id         int32     `json:""`
	Name       string    `json:""`
	Size       [2]uint64 `json:""`
	Format     uint16    `json:""`
	Type       uint16    `json:""`
	Compressed uint16    `json:""`
	Data       []byte    `json:"-"`
	Repeated   bool
}

type BaseMaterial struct {
	Color        [3]byte `json:"color"`
	Transparency float32 `json:"transparency"`
}

func (m *BaseMaterial) HasTexture() bool {
	return false
}

type TextureMaterial struct {
	BaseMaterial
	Texture *Texture `json:"texture,omitempty"`
	Normal  *Texture `json:"normal,omitempty"`
}

func (m *TextureMaterial) HasTexture() bool {
	return m.Texture != nil
}

type PbrMaterial struct {
	TextureMaterial
	Emissive            [4]byte `json:"emissive"`
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

type MeshTriangle struct {
	Batchid int32       `json:"batchid"`
	Faces   [][3]uint32 `json:"faces"`
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
}

func (nd *MeshNode) GetBoundbox() *Extent3 {
	ext := NewExtent3()
	for i := range nd.Vertices {
		ext.AddPoints([]float64{float64(nd.Vertices[i][0]), float64(nd.Vertices[i][1]), float64(nd.Vertices[i][2])})
	}
	return ext
}

type Mesh struct {
	Version   uint32         `json:"version"`
	Materials []MeshMaterial `json:"materials,omitempty"`
	Nodes     []*MeshNode    `json:"nodes,omitempty"`
}

func NewMesh() *Mesh {
	return &Mesh{Version: 1}
}

func (m *Mesh) NodeCount() int {
	return len(m.Nodes)
}

func (m *Mesh) MaterialCount() int {
	return len(m.Materials)
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

func readLittleByte(rd io.Reader, v interface{}) {
	binary.Read(rd, binary.LittleEndian, v)
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
	nm := make([]byte, name_size, name_size)
	rd.Read(nm)
	tex.Name = string(nm)
	readLittleByte(rd, &tex.Size)
	readLittleByte(rd, &tex.Format)
	readLittleByte(rd, &tex.Type)
	readLittleByte(rd, &tex.Compressed)
	var tex_size uint32
	readLittleByte(rd, &tex_size)
	tex.Data = make([]byte, tex_size, tex_size)
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

func PbrMaterialMarshal(wt io.Writer, mtl *PbrMaterial) {
	TextureMaterialMarshal(wt, &mtl.TextureMaterial)
	writeLittleByte(wt, mtl.Emissive[:])
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

func PbrMaterialUnMarshal(rd io.Reader) *PbrMaterial {
	mtl := PbrMaterial{}
	tmtl := TextureMaterialUnMarshal(rd)
	mtl.TextureMaterial = *tmtl
	readLittleByte(rd, mtl.Emissive[:])
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

func MaterialMarshal(wt io.Writer, mt MeshMaterial) {
	switch mtl := mt.(type) {
	case *BaseMaterial:
		writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_COLOR))
		BaseMaterialMarshal(wt, mtl)
		break
	case *TextureMaterial:
		writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_TEXTURE))
		TextureMaterialMarshal(wt, mtl)
		break
	case *PbrMaterial:
		writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_PBR))
		PbrMaterialMarshal(wt, mtl)
		break
	case *LambertMaterial:
		writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_LAMBERT))
		LambertMaterialMarshal(wt, mtl)
		break
	case *PhongMaterial:
		writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_PHONG))
		PhongMaterialMarshal(wt, mtl)
		break
	default:
		break
	}
}

func MaterialUnMarshal(rd io.Reader) MeshMaterial {
	var ty uint32
	readLittleByte(rd, &ty)
	switch int(ty) {
	case MESH_TRIANGLE_MATERIAL_TYPE_COLOR:
		return BaseMaterialUnMarshal(rd)
	case MESH_TRIANGLE_MATERIAL_TYPE_TEXTURE:
		return TextureMaterialUnMarshal(rd)
	case MESH_TRIANGLE_MATERIAL_TYPE_PBR:
		return PbrMaterialUnMarshal(rd)
	case MESH_TRIANGLE_MATERIAL_TYPE_LAMBERT:
		return LambertMaterialUnMarshal(rd)
	case MESH_TRIANGLE_MATERIAL_TYPE_PHONG:
		return PhongMaterialUnMarshal(rd)
	default:
		return nil
	}
}

func MtlsMarshal(wt io.Writer, mtls []MeshMaterial) {
	writeLittleByte(wt, uint32(len(mtls)))
	for _, mtl := range mtls {
		MaterialMarshal(wt, mtl)
	}
}

func MtlsUnMarshal(rd io.Reader) []MeshMaterial {
	var size uint32
	readLittleByte(rd, &size)
	mtls := make([]MeshMaterial, size, size)
	for i := 0; i < int(size); i++ {
		mtls[i] = MaterialUnMarshal(rd)
	}
	return mtls
}

func MeshTriangleMarshal(wt io.Writer, nd *MeshTriangle) {
	writeLittleByte(wt, nd.Batchid)
	writeLittleByte(wt, uint32(len(nd.Faces)))
	for _, f := range nd.Faces {
		writeLittleByte(wt, &f)
	}
}

func MeshTriangleUnMarshal(rd io.Reader) *MeshTriangle {
	nd := MeshTriangle{}
	readLittleByte(rd, &nd.Batchid)
	var size uint32
	readLittleByte(rd, &size)
	nd.Faces = make([][3]uint32, size, size)
	for i := 0; i < int(size); i++ {
		readLittleByte(rd, &nd.Faces[i])
	}
	return &nd
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
	nd.Edges = make([][2]uint32, size, size)
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
	return &nd
}

func MeshNodesMarshal(wt io.Writer, nds []*MeshNode) {
	writeLittleByte(wt, uint32(len(nds)))
	for _, nd := range nds {
		MeshNodeMarshal(wt, nd)
	}
}

func MeshNodesUnMarshal(rd io.Reader) []*MeshNode {
	var size uint32
	readLittleByte(rd, &size)
	nds := make([]*MeshNode, size, size)
	for i := range nds {
		nds[i] = MeshNodeUnMarshal(rd)
	}
	return nds
}

func MeshMarshal(wt io.Writer, ms *Mesh) {
	wt.Write([]byte(MESH_SIGNATURE))
	writeLittleByte(wt, ms.Version)
	MtlsMarshal(wt, ms.Materials)
	MeshNodesMarshal(wt, ms.Nodes)
}

func MeshUnMarshal(rd io.Reader) *Mesh {
	ms := Mesh{}
	sig := make([]byte, 4, 4)
	rd.Read(sig)
	readLittleByte(rd, &ms.Version)
	ms.Materials = MtlsUnMarshal(rd)
	ms.Nodes = MeshNodesUnMarshal(rd)
	return &ms
}

func MeshReadFrom(path string) (*Mesh, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	return MeshUnMarshal(f), nil
}

func MeshWriteTo(path string, ms *Mesh) error {
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
	f, e := os.Create(path)
	if e != nil {
		return e
	}
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
	return ioutil.ReadAll(r)
}

func LoadTexture(tex *Texture, flipY bool) (image.Image, error) {
	w := int(tex.Size[0])
	h := int(tex.Size[1])
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	data := tex.Data
	var sz int
	if tex.Format == TEXTURE_FORMAT_RGB {
		sz = 3
	} else if tex.Format == TEXTURE_FORMAT_RGBA {
		sz = 4
	} else if tex.Format == TEXTURE_FORMAT_R {
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
			if sz == 4 {
				c = color.NRGBA{R: data[p], G: data[p+1], B: data[p+2], A: data[p+3]}
			} else if sz == 3 {
				c = color.NRGBA{R: data[p], G: data[p+1], B: data[p+2], A: 255}
			} else if sz == 1 {
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