package mst

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"

	dmat "github.com/flywave/go3d/float64/mat4"

	"github.com/flywave/go3d/vec2"
	"github.com/flywave/go3d/vec3"
)

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
