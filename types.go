package mst

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

// MeshMaterial 接口定义了材质的基本方法
type MeshMaterial interface {
	HasTexture() bool
	GetTexture() *Texture
	GetColor() [3]byte
	GetEmissive() [3]byte
}

// Face 面结构
type Face struct {
	Vertex [3]uint32
	Normal *[3]uint32
	Uv     *[3]uint32
}

// MeshTriangle 网格三角形
type MeshTriangle struct {
	Batchid int32   `json:"batchid"`
	Faces   []*Face `json:"faces"`
}

// MeshOutline 网格轮廓
type MeshOutline struct {
	Batchid int32       `json:"batchid"`
	Edges   [][2]uint32 `json:"edges"`
}
