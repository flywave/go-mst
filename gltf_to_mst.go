package mst

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"

	dmat "github.com/flywave/go3d/float64/mat4"
	"github.com/flywave/go3d/float64/vec4"

	"github.com/flywave/go3d/vec2"
	"github.com/flywave/go3d/vec3"

	"github.com/qmuntal/gltf"
)

var (
	emptyMatrix = [16]float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

type GltfToMst struct {
}

func (g *GltfToMst) Convert(path string) (*Mesh, []*InstanceMst, *[6]float64, error) {
	mesh := NewMesh()
	bbx := &[6]float64{}
	doc, err := gltf.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}

	isInstance := make(map[uint32]bool)
	for _, nd := range doc.Nodes {
		if _, ok := isInstance[*nd.Mesh]; ok {
			isInstance[*nd.Mesh] = true
		} else {
			isInstance[*nd.Mesh] = false
		}
	}

	instMp := make(map[uint32]*InstanceMst)
	for _, nd := range doc.Nodes {
		meshId := *nd.Mesh
		if v := isInstance[meshId]; v {
			bx := g.transMesh(doc, mesh, &doc.Meshes[meshId])

			addPoint(bbx, &[3]float64{bx[0], bx[1], bx[2]})
			addPoint(bbx, &[3]float64{bx[3], bx[4], bx[5]})
		} else {
			var inst *InstanceMst
			var ok bool
			if inst, ok = instMp[meshId]; ok {
				insMesh := NewMesh()
				bx := g.transMesh(doc, insMesh, &doc.Meshes[meshId])
				inst = &InstanceMst{Mst: insMesh, BBox: bx}
				instMp[meshId] = inst
			}
			inst.Transfors = append(inst.Transfors, toMat(nd.Matrix))
		}
	}
	insts := []*InstanceMst{}
	for _, v := range instMp {
		insts = append(insts, v)
	}
	return mesh, insts, bbx, nil
}

func (g *GltfToMst) transMesh(doc *gltf.Document, mstMh *Mesh, mh *gltf.Mesh) *[6]float64 {
	accMap := make(map[uint32]bool)
	mhNode := &MeshNode{}
	bbx := &[6]float64{}
	var faceBuff *gltf.Buffer
	var posBuff *gltf.Buffer
	var posView *gltf.BufferView
	var texBuff *gltf.Buffer
	var texView *gltf.BufferView
	var nlBuff *gltf.Buffer
	var nlView *gltf.BufferView
	for _, ps := range mh.Primitives {
		tg := &MeshTriangle{}
		tg.Batchid = int32(len(mstMh.Materials))
		g.transMaterial(doc, mstMh, *ps.Material)
		acc := doc.Accessors[int(*ps.Indices)]
		faceBuff = &doc.Buffers[int(doc.BufferViews[int(*acc.BufferView)].Buffer)]
		tg.Faces = make([][3]uint32, int(acc.Count/3))
		bytePerIndices := 1
		if acc.ComponentType == gltf.Short || acc.ComponentType == gltf.UnsignedShort {
			bytePerIndices = 2
		} else if acc.ComponentType == gltf.UnsignedInt || acc.ComponentType == gltf.Float {
			bytePerIndices = 4
		}
		bf := bytes.NewBuffer(faceBuff.Data[acc.ByteOffset : int(acc.ByteOffset)+int(acc.Count)*bytePerIndices])
		for i := 0; i < len(tg.Faces); i++ {
			binary.Read(bf, binary.LittleEndian, &tg.Faces[i])
		}

		if idx, ok := ps.Attributes["POSITION"]; ok {
			if _, ok := accMap[idx]; !ok {
				acc = doc.Accessors[idx]
				posView = &doc.BufferViews[int(*acc.BufferView)]
				posBuff = &doc.Buffers[int(posView.Buffer)]
				bf := bytes.NewBuffer(posBuff.Data[int(posView.ByteOffset):int(posView.ByteOffset+posView.ByteLength)])
				for i := 0; i < int(acc.Count); i++ {
					v := vec3.T{}
					binary.Read(bf, binary.LittleEndian, &v)
					mhNode.Vertices = append(mhNode.Vertices, v)
					addPoint(bbx, &[3]float64{float64(v[0]), float64(v[1]), float64(v[2])})
				}
				accMap[idx] = true
			}
		}

		if idx, ok := ps.Attributes["TEXCOORD_0"]; ok {
			if _, ok := accMap[idx]; !ok {
				acc = doc.Accessors[idx]
				texView = &doc.BufferViews[int(*acc.BufferView)]
				texBuff = &doc.Buffers[int(texView.Buffer)]
				bf := bytes.NewBuffer(texBuff.Data[int(texView.ByteOffset):int(texView.ByteOffset+texView.ByteLength)])
				for i := 0; i < int(acc.Count); i++ {
					v := vec2.T{}
					binary.Read(bf, binary.LittleEndian, &v)
					mhNode.TexCoords = append(mhNode.TexCoords, v)
				}
				accMap[idx] = true
			}
		}

		if idx, ok := ps.Attributes["NORMAL"]; ok {
			if _, ok := accMap[idx]; !ok {
				acc = doc.Accessors[idx]
				nlView = &doc.BufferViews[int(*acc.BufferView)]
				nlBuff = &doc.Buffers[int(nlView.Buffer)]
				bf := bytes.NewBuffer(nlBuff.Data[int(nlView.ByteOffset):int(nlView.ByteOffset+texView.ByteLength)])
				for i := 0; i < int(acc.Count); i++ {
					v := vec3.T{}
					binary.Read(bf, binary.LittleEndian, &v)
					mhNode.Normals = append(mhNode.Normals, v)
				}
				accMap[idx] = true
			}
		}
		mhNode.FaceGroup = append(mhNode.FaceGroup, tg)
	}
	mstMh.Nodes = append(mstMh.Nodes, mhNode)
	return bbx
}

func (g *GltfToMst) transMaterial(doc *gltf.Document, mstMh *Mesh, id uint32) {
	mt := doc.Materials[id]
	mtl := &PbrMaterial{}
	mtl.Emissive[0] = byte(mt.EmissiveFactor[0] * 255)
	mtl.Emissive[1] = byte(mt.EmissiveFactor[0] * 255)
	mtl.Emissive[2] = byte(mt.EmissiveFactor[0] * 255)
	if mt.PBRMetallicRoughness.BaseColorFactor != nil {
		mtl.Color[0] = byte(mt.PBRMetallicRoughness.BaseColorFactor.R * 255)
		mtl.Color[1] = byte(mt.PBRMetallicRoughness.BaseColorFactor.G * 255)
		mtl.Color[2] = byte(mt.PBRMetallicRoughness.BaseColorFactor.B * 255)
		mtl.Transparency = 1 - float32(mt.PBRMetallicRoughness.BaseColorFactor.A)
	}
	if mt.PBRMetallicRoughness.MetallicFactor != nil {
		mtl.Metallic = float32(*mt.PBRMetallicRoughness.MetallicFactor)
	}
	if mt.PBRMetallicRoughness.RoughnessFactor != nil {
		mtl.Roughness = float32(*mt.PBRMetallicRoughness.RoughnessFactor)
	}
	if mt.PBRMetallicRoughness.BaseColorTexture != nil {
		texInfo := mt.PBRMetallicRoughness.BaseColorTexture
		texIdx := texInfo.Index
		src := *doc.Textures[int(texIdx)].Source
		img := doc.Images[int(src)]
		var tex *Texture
		var buf io.Reader
		var err error
		if img.BufferView != nil {
			view := doc.BufferViews[int(*img.BufferView)]
			bufferIdx := view.Buffer
			buffer := doc.Buffers[int(bufferIdx)]
			bt := buffer.Data[view.ByteOffset : view.ByteOffset+view.ByteLength]
			buf = bytes.NewBuffer(bt)
		}
		tex, err = g.decodeImage(img.MimeType, buf)
		if err != nil {
			return
		}
		if tex != nil {
			tex.Id = int32(texIdx)
			mtl.TextureMaterial.Texture = tex
		}
	}
	mstMh.Materials = append(mstMh.Materials, mtl)
}

func (g *GltfToMst) decodeImage(mime string, rd io.Reader) (*Texture, error) {
	var img image.Image
	var err error
	tex := &Texture{}
	if mime == "image/png" {
		img, err = png.Decode(rd)
	} else if mime == "image/jpg" || mime == "image/jpeg" {
		img, err = jpeg.Decode(rd)
	}
	if err != nil {
		return nil, err
	}
	if img != nil {
		w := img.Bounds().Size().X
		h := img.Bounds().Size().Y
		tex.Size[0] = uint64(w)
		tex.Size[1] = uint64(h)
		var buf []byte
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				cl := img.At(x, y)
				r, g, b, a := color.RGBAModel.Convert(cl).RGBA()
				buf = append(buf, byte(r), byte(g), byte(b), byte(a))
			}
		}
		tex.Format = TEXTURE_FORMAT_RGBA
		tex.Compressed = TEXTURE_COMPRESSED_ZLIB
		tex.Data = CompressImage(buf)
		return tex, nil
	}
	return nil, errors.New("not support image type")
}

func toMat(mat [16]float64) *dmat.T {
	m := &dmat.T{}
	m[0] = vec4.T{mat[0], mat[1], mat[2], mat[3]}
	m[1] = vec4.T{mat[4], mat[5], mat[6], mat[7]}
	m[2] = vec4.T{mat[8], mat[9], mat[10], mat[11]}
	m[3] = vec4.T{mat[12], mat[13], mat[14], mat[15]}
	return m
}

func addPoint(bx *[6]float64, p *[3]float64) {
	bx[0] = math.Min(bx[0], p[0])
	bx[1] = math.Min(bx[1], p[1])
	bx[2] = math.Min(bx[2], p[2])

	bx[3] = math.Max(bx[3], p[0])
	bx[4] = math.Max(bx[4], p[1])
	bx[5] = math.Max(bx[5], p[2])
}
