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

func writeLittleByte(wt io.Writer, v interface{}) error {
	buf := toLittleByteOrder(v)
	if buf != nil {
		_, err := wt.Write(buf)
		return err
	}
	return nil
}

func readLittleByte(rd io.Reader, v interface{}) error {
	return binary.Read(rd, binary.LittleEndian, v)
}

func BaseMaterialMarshal(wt io.Writer, mtl *BaseMaterial) error {
	if err := writeLittleByte(wt, &mtl.Color); err != nil {
		return err
	}
	return writeLittleByte(wt, &mtl.Transparency)
}

func BaseMaterialUnMarshal(rd io.Reader) *BaseMaterial {
	mtl := BaseMaterial{}
	readLittleByte(rd, mtl.Color[:])
	readLittleByte(rd, &mtl.Transparency)
	return &mtl
}

func TextureMarshal(wt io.Writer, tex *Texture) error {
	if err := writeLittleByte(wt, tex.Id); err != nil {
		return err
	}
	if err := writeLittleByte(wt, uint32(len(tex.Name))); err != nil {
		return err
	}
	if _, err := wt.Write([]byte(tex.Name)); err != nil {
		return err
	}
	if err := writeLittleByte(wt, &tex.Size); err != nil {
		return err
	}
	if err := writeLittleByte(wt, tex.Format); err != nil {
		return err
	}
	if err := writeLittleByte(wt, tex.Type); err != nil {
		return err
	}
	if err := writeLittleByte(wt, tex.Compressed); err != nil {
		return err
	}
	if err := writeLittleByte(wt, uint32(len(tex.Data))); err != nil {
		return err
	}
	if _, err := wt.Write(tex.Data); err != nil {
		return err
	}
	return writeLittleByte(wt, tex.Repeated)
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

func TextureMaterialMarshal(wt io.Writer, mtl *TextureMaterial) error {
	if err := BaseMaterialMarshal(wt, &mtl.BaseMaterial); err != nil {
		return err
	}
	if mtl.Texture != nil {
		if err := writeLittleByte(wt, uint16(1)); err != nil {
			return err
		}
		if err := TextureMarshal(wt, mtl.Texture); err != nil {
			return err
		}
	} else {
		if err := writeLittleByte(wt, uint16(0)); err != nil {
			return err
		}
	}
	if mtl.Normal != nil {
		if err := writeLittleByte(wt, uint16(1)); err != nil {
			return err
		}
		return TextureMarshal(wt, mtl.Normal)
	} else {
		return writeLittleByte(wt, uint16(0))
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

func PbrMaterialMarshal(wt io.Writer, mtl *PbrMaterial, v uint32) error {
	if err := TextureMaterialMarshal(wt, &mtl.TextureMaterial); err != nil {
		return err
	}
	if err := writeLittleByte(wt, mtl.Emissive[:]); err != nil {
		return err
	}
	if v < V2 {
		if err := writeLittleByte(wt, byte(255)); err != nil {
			return err
		}
	}
	if err := writeLittleByte(wt, &mtl.Metallic); err != nil {
		return err
	}
	if err := writeLittleByte(wt, &mtl.Roughness); err != nil {
		return err
	}
	if err := writeLittleByte(wt, &mtl.Reflectance); err != nil {
		return err
	}
	if err := writeLittleByte(wt, &mtl.AmbientOcclusion); err != nil {
		return err
	}
	if err := writeLittleByte(wt, &mtl.ClearCoat); err != nil {
		return err
	}
	if err := writeLittleByte(wt, &mtl.ClearCoatRoughness); err != nil {
		return err
	}
	if err := writeLittleByte(wt, mtl.ClearCoatNormal[:]); err != nil {
		return err
	}
	if err := writeLittleByte(wt, &mtl.Anisotropy); err != nil {
		return err
	}
	if err := writeLittleByte(wt, mtl.AnisotropyDirection[:]); err != nil {
		return err
	}
	if err := writeLittleByte(wt, &mtl.Thickness); err != nil {
		return err
	}
	if err := writeLittleByte(wt, &mtl.SubSurfacePower); err != nil {
		return err
	}
	if err := writeLittleByte(wt, mtl.SheenColor[:]); err != nil {
		return err
	}
	return writeLittleByte(wt, mtl.SubSurfaceColor[:])
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

func LambertMaterialMarshal(wt io.Writer, mtl *LambertMaterial) error {
	if err := TextureMaterialMarshal(wt, &mtl.TextureMaterial); err != nil {
		return err
	}
	if err := writeLittleByte(wt, mtl.Ambient[:]); err != nil {
		return err
	}
	if err := writeLittleByte(wt, mtl.Diffuse[:]); err != nil {
		return err
	}
	return writeLittleByte(wt, mtl.Emissive[:])
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

func PhongMaterialMarshal(wt io.Writer, mtl *PhongMaterial) error {
	if err := LambertMaterialMarshal(wt, &mtl.LambertMaterial); err != nil {
		return err
	}
	if err := writeLittleByte(wt, mtl.Specular[:]); err != nil {
		return err
	}
	if err := writeLittleByte(wt, &mtl.Shininess); err != nil {
		return err
	}
	return writeLittleByte(wt, &mtl.Specularity)
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

func MaterialMarshal(wt io.Writer, mt MeshMaterial, v uint32) error {
	switch mtl := mt.(type) {
	case *BaseMaterial:
		if err := writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_COLOR)); err != nil {
			return err
		}
		return BaseMaterialMarshal(wt, mtl)
	case *TextureMaterial:
		if err := writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_TEXTURE)); err != nil {
			return err
		}
		return TextureMaterialMarshal(wt, mtl)
	case *PbrMaterial:
		if err := writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_PBR)); err != nil {
			return err
		}
		return PbrMaterialMarshal(wt, mtl, v)
	case *LambertMaterial:
		if err := writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_LAMBERT)); err != nil {
			return err
		}
		return LambertMaterialMarshal(wt, mtl)
	case *PhongMaterial:
		if err := writeLittleByte(wt, uint32(MESH_TRIANGLE_MATERIAL_TYPE_PHONG)); err != nil {
			return err
		}
		return PhongMaterialMarshal(wt, mtl)
	}
	return nil
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

func MtlsMarshal(wt io.Writer, mtls []MeshMaterial, v uint32) error {
	if err := writeLittleByte(wt, uint32(len(mtls))); err != nil {
		return err
	}
	for _, mtl := range mtls {
		if err := MaterialMarshal(wt, mtl, v); err != nil {
			return err
		}
	}
	return nil
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

func MeshTriangleMarshal(wt io.Writer, nd *MeshTriangle) error {
	if err := writeLittleByte(wt, nd.Batchid); err != nil {
		return err
	}
	if err := writeLittleByte(wt, uint32(len(nd.Faces))); err != nil {
		return err
	}
	for _, f := range nd.Faces {
		if err := writeLittleByte(wt, &f.Vertex); err != nil {
			return err
		}
	}
	return nil
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

func MeshOutlineMarshal(wt io.Writer, nd *MeshOutline) error {
	if err := writeLittleByte(wt, nd.Batchid); err != nil {
		return err
	}
	if err := writeLittleByte(wt, uint32(len(nd.Edges))); err != nil {
		return err
	}
	for _, e := range nd.Edges {
		if err := writeLittleByte(wt, &e); err != nil {
			return err
		}
	}
	return nil
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

func MeshNodeMarshal(wt io.Writer, nd *MeshNode) error {
	if err := writeLittleByte(wt, uint32(len(nd.Vertices))); err != nil {
		return err
	}
	for i := range nd.Vertices {
		if err := writeLittleByte(wt, nd.Vertices[i][:]); err != nil {
			return err
		}
	}
	if err := writeLittleByte(wt, uint32(len(nd.Normals))); err != nil {
		return err
	}
	for i := range nd.Normals {
		if err := writeLittleByte(wt, nd.Normals[i][:]); err != nil {
			return err
		}
	}
	if err := writeLittleByte(wt, uint32(len(nd.Colors))); err != nil {
		return err
	}
	for i := range nd.Colors {
		if err := writeLittleByte(wt, nd.Colors[i][:]); err != nil {
			return err
		}
	}
	if err := writeLittleByte(wt, uint32(len(nd.TexCoords))); err != nil {
		return err
	}
	for i := range nd.TexCoords {
		if err := writeLittleByte(wt, nd.TexCoords[i][:]); err != nil {
			return err
		}
	}
	if nd.Mat != nil {
		if err := writeLittleByte(wt, uint8(1)); err != nil {
			return err
		}
		if err := writeLittleByte(wt, nd.Mat[0][:]); err != nil {
			return err
		}
		if err := writeLittleByte(wt, nd.Mat[1][:]); err != nil {
			return err
		}
		if err := writeLittleByte(wt, nd.Mat[2][:]); err != nil {
			return err
		}
		if err := writeLittleByte(wt, nd.Mat[3][:]); err != nil {
			return err
		}
	} else {
		if err := writeLittleByte(wt, uint8(0)); err != nil {
			return err
		}
	}

	if err := writeLittleByte(wt, uint32(len(nd.FaceGroup))); err != nil {
		return err
	}
	for _, fg := range nd.FaceGroup {
		if err := MeshTriangleMarshal(wt, fg); err != nil {
			return err
		}
	}

	if err := writeLittleByte(wt, uint32(len(nd.EdgeGroup))); err != nil {
		return err
	}
	for _, eg := range nd.EdgeGroup {
		if err := MeshOutlineMarshal(wt, eg); err != nil {
			return err
		}
	}
	return nil
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

func MeshNodesMarshal(wt io.Writer, nds []*MeshNode) error {
	if err := writeLittleByte(wt, uint32(len(nds))); err != nil {
		return err
	}
	for _, nd := range nds {
		if err := MeshNodeMarshal(wt, nd); err != nil {
			return err
		}
	}
	return nil
}

func MeshNodesMarshalWithVersion(wt io.Writer, nds []*MeshNode, v uint32) error {
	if err := writeLittleByte(wt, uint32(len(nds))); err != nil {
		return err
	}
	for _, nd := range nds {
		if v >= V5 {
			if err := MeshNodeMarshal(wt, nd); err != nil {
				return err
			}
		} else {
			if err := MeshNodeMarshalWithoutProps(wt, nd); err != nil {
				return err
			}
		}
	}
	return nil
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

func MeshMarshal(wt io.Writer, ms *Mesh) error {
	if _, err := wt.Write([]byte(MESH_SIGNATURE)); err != nil {
		return err
	}
	if err := writeLittleByte(wt, ms.Version); err != nil {
		return err
	}
	// V4及以上版本序列化Code字段
	if ms.Version >= V4 {
		if err := writeLittleByte(wt, ms.BaseMesh.Code); err != nil {
			return err
		}
	}
	if err := MtlsMarshal(wt, ms.Materials, ms.Version); err != nil {
		return err
	}
	if err := MeshNodesMarshalWithVersion(wt, ms.Nodes, ms.Version); err != nil {
		return err
	}
	if err := MeshInstanceNodesMarshal(wt, ms.InstanceNode, ms.Version); err != nil {
		return err
	}
	// V5 版本序列化新增属性
	if ms.Version >= V5 {
		if ms.Props != nil && len(*ms.Props) > 0 {
			// 先写入标记位1表示有Properties
			if err := writeLittleByte(wt, uint32(1)); err != nil {
				return err
			}
			return PropertiesMarshal(wt, ms.Props)
		} else {
			// 如果Props为nil，写入标记位0
			if err := writeLittleByte(wt, uint32(0)); err != nil {
				return err
			}
		}
	}
	return nil
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

func MeshInstanceNodesMarshal(wt io.Writer, instNd []*InstanceMesh, v uint32) error {
	if err := writeLittleByte(wt, uint32(len(instNd))); err != nil {
		return err
	}
	for _, nd := range instNd {
		if err := MeshInstanceNodeMarshal(wt, nd, v); err != nil {
			return err
		}
	}
	return nil
}

// MeshNodesMarshalForInstanceMesh 序列化InstanceMesh中的MeshNode，不序列化Props属性
func MeshNodesMarshalForInstanceMesh(wt io.Writer, nds []*MeshNode) error {
	if err := writeLittleByte(wt, uint32(len(nds))); err != nil {
		return err
	}
	for _, nd := range nds {
		if err := MeshNodeMarshalWithoutProps(wt, nd); err != nil {
			return err
		}
	}
	return nil
}

// MeshNodeMarshalWithoutProps 序列化MeshNode，不序列化Props属性
func MeshNodeMarshalWithoutProps(wt io.Writer, nd *MeshNode) error {
	if err := writeLittleByte(wt, uint32(len(nd.Vertices))); err != nil {
		return err
	}
	for i := range nd.Vertices {
		if err := writeLittleByte(wt, nd.Vertices[i][:]); err != nil {
			return err
		}
	}
	if err := writeLittleByte(wt, uint32(len(nd.Normals))); err != nil {
		return err
	}
	for i := range nd.Normals {
		if err := writeLittleByte(wt, nd.Normals[i][:]); err != nil {
			return err
		}
	}
	if err := writeLittleByte(wt, uint32(len(nd.Colors))); err != nil {
		return err
	}
	for i := range nd.Colors {
		if err := writeLittleByte(wt, nd.Colors[i][:]); err != nil {
			return err
		}
	}
	if err := writeLittleByte(wt, uint32(len(nd.TexCoords))); err != nil {
		return err
	}
	for i := range nd.TexCoords {
		if err := writeLittleByte(wt, nd.TexCoords[i][:]); err != nil {
			return err
		}
	}
	if nd.Mat != nil {
		if err := writeLittleByte(wt, uint8(1)); err != nil {
			return err
		}
		if err := writeLittleByte(wt, nd.Mat[0][:]); err != nil {
			return err
		}
		if err := writeLittleByte(wt, nd.Mat[1][:]); err != nil {
			return err
		}
		if err := writeLittleByte(wt, nd.Mat[2][:]); err != nil {
			return err
		}
		if err := writeLittleByte(wt, nd.Mat[3][:]); err != nil {
			return err
		}
	} else {
		if err := writeLittleByte(wt, uint8(0)); err != nil {
			return err
		}
	}

	if err := writeLittleByte(wt, uint32(len(nd.FaceGroup))); err != nil {
		return err
	}
	for _, fg := range nd.FaceGroup {
		if err := MeshTriangleMarshal(wt, fg); err != nil {
			return err
		}
	}

	if err := writeLittleByte(wt, uint32(len(nd.EdgeGroup))); err != nil {
		return err
	}
	for _, eg := range nd.EdgeGroup {
		if err := MeshOutlineMarshal(wt, eg); err != nil {
			return err
		}
	}
	return nil
}

func MeshInstanceNodeMarshal(wt io.Writer, instNd *InstanceMesh, v uint32) error {
	if err := writeLittleByte(wt, uint32(len(instNd.Transfors))); err != nil {
		return err
	}
	for _, mt := range instNd.Transfors {
		if err := writeLittleByte(wt, mt[0][:]); err != nil {
			return err
		}
		if err := writeLittleByte(wt, mt[1][:]); err != nil {
			return err
		}
		if err := writeLittleByte(wt, mt[2][:]); err != nil {
			return err
		}
		if err := writeLittleByte(wt, mt[3][:]); err != nil {
			return err
		}
	}
	if err := writeLittleByte(wt, uint32(len(instNd.Features))); err != nil {
		return err
	}
	for _, f := range instNd.Features {
		if err := writeLittleByte(wt, f); err != nil {
			return err
		}
	}
	if err := writeLittleByte(wt, instNd.BBox); err != nil {
		return err
	}
	// 序列化Mesh字段
	if err := MtlsMarshal(wt, instNd.Mesh.Materials, v); err != nil {
		return err
	}
	// 修复：使用正确的函数来序列化Mesh.Nodes，确保Props字段能被正确处理
	// 对于InstanceMesh中的Mesh.Nodes，我们不应该序列化Props属性，因为Props是InstanceMesh的独立属性
	if err := MeshNodesMarshalForInstanceMesh(wt, instNd.Mesh.Nodes); err != nil {
		return err
	}
	// V4及以上版本序列化Code字段
	if v >= V4 {
		if err := writeLittleByte(wt, instNd.Mesh.Code); err != nil {
			return err
		}
	}
	// V5 版本序列化新增属性
	if v >= V5 {
		// 确保Props数组的数量与Features和Transfors对应
		expectedLen := len(instNd.Transfors)
		if len(instNd.Features) > expectedLen {
			expectedLen = len(instNd.Features)
		}

		// 写入Props数组长度
		if err := writeLittleUint32(wt, uint32(expectedLen)); err != nil {
			return err
		}

		// 写入每个Props元素
		for i := 0; i < expectedLen; i++ {
			var props *Properties
			if instNd.Props != nil && i < len(instNd.Props) {
				props = instNd.Props[i]
			}

			if props != nil && len(*props) > 0 {
				// 写入标记位1表示有Properties
				if err := writeLittleUint32(wt, uint32(1)); err != nil {
					return err
				}
				if err := PropertiesMarshal(wt, props); err != nil {
					return err
				}
			} else {
				// 如果Props为nil或空，写入标记位0
				if err := writeLittleUint32(wt, uint32(0)); err != nil {
					return err
				}
			}
		}
	}
	return writeLittleByte(wt, instNd.Hash)
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
		// 读取Props数组长度
		var propsLen uint32
		if err := readLittleByte(rd, &propsLen); err != nil {
			return nil
		}

		// 确保Props数组的数量与Features和Transfors对应
		expectedLen := len(inst.Transfors)
		if len(inst.Features) > expectedLen {
			expectedLen = len(inst.Features)
		}
		if int(propsLen) > expectedLen {
			expectedLen = int(propsLen)
		}

		// 创建Props数组
		inst.Props = make([]*Properties, expectedLen)

		// 读取每个Props元素
		for i := 0; i < int(propsLen); i++ {
			var hasProps uint32
			if err := readLittleByte(rd, &hasProps); err != nil {
				return nil
			}

			if hasProps > 0 {
				props := PropertiesUnMarshal(rd)
				if props == nil {
					return nil
				}
				inst.Props[i] = props
			} else {
				inst.Props[i] = nil
			}
		}

		// 对于超出propsLen但小于expectedLen的部分，设置为nil
		for i := int(propsLen); i < expectedLen; i++ {
			inst.Props[i] = nil
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
	return MeshMarshal(f, ms)
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

	return &nd
}
