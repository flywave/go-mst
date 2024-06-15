package mst

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	proj "github.com/flywave/go-proj"
	"github.com/flywave/go3d/float64/vec3"
	fvec3 "github.com/flywave/go3d/vec3"
	"github.com/qmuntal/gltf"
	"github.com/xtgo/uuid"
)

func TestGltf3(t *testing.T) {
	f, _ := os.Open("./tests/aa74a4e312afeae291f11dabcb5098d3.mst")
	mh := MeshUnMarshal(f)
	mh.InstanceNode = nil
	doc := CreateDoc()
	BuildGltf(doc, mh, false, false)
	bt, _ := GetGltfBinary(doc, 8)
	ioutil.WriteFile("./tests/aa74a4e312afeae291f11dabcb5098d3.mst.glb", bt, os.ModePerm)
}

func TestPipe2(t *testing.T) {
	id := uuid.NewRandom().String()

	id = strings.ReplaceAll(id, "-", "")
	fmt.Println(id)
}

func TestDe(t *testing.T) {
	gltf.Open("/home/hj/workspace/GISCore/build/public/Resources/anchormodel/public/psqitong/psqitong.glb")
	gltf.Open("/home/hj/workspace/GISCore/build/public/Resources/anchormodel/public/psqitong/qitong.glb")
}

func MstToObj(path, destName string) {
	dir, _ := filepath.Split(path)
	faceTemp1 := "f %d %d %d \n"
	faceTemp12 := "f %d//%d %d//%d %d//%d \n"
	faceTemp21 := "f %d/%d %d/%d %d/%d \n"

	faceTemp3 := "f %d/%d/%d %d/%d/%d %d/%d/%d \n"

	vertTmp := "v %f %f %f \n"
	nvTmp := "vn %f %f %f \n"

	uvTmp := "vt %f %f \n"

	ms, _ := MeshReadFrom(path)
	fl, _ := os.Create(fmt.Sprintf("%s/%s_convert.obj", dir, destName))
	mtlTex, _ := os.Create(fmt.Sprintf("%s/%s_convert.mtl", dir, destName))
	fl.Write([]byte(fmt.Sprintf("mtllib %s_convert.mtl \n", destName)))
	var vertCount uint32 = 1
	for _, nd := range ms.Nodes {
		for _, v := range nd.Vertices {
			fl.Write([]byte(fmt.Sprintf(vertTmp, v[0], v[1], v[2])))
		}
		for _, v := range nd.Normals {
			fl.Write([]byte(fmt.Sprintf(nvTmp, v[0], v[1], v[2])))
		}
		for _, v := range nd.TexCoords {
			fl.Write([]byte(fmt.Sprintf(uvTmp, v[0], v[1])))
		}
	}

	for _, nd := range ms.Nodes {
		hasvn := false
		hasvt := false
		if len(nd.Normals) > 0 {
			hasvn = true
		}
		if len(nd.TexCoords) > 0 {
			hasvt = true
		}
		for _, g := range nd.FaceGroup {
			fl.Write([]byte(fmt.Sprintf("usemtl material_%d \n", g.Batchid)))
			if hasvn && hasvt {
				for _, face := range g.Faces {
					fl.Write([]byte(fmt.Sprintf(faceTemp3, face.Vertex[0]+vertCount, face.Vertex[0]+vertCount, face.Vertex[0]+vertCount, face.Vertex[1]+vertCount, face.Vertex[1]+vertCount, face.Vertex[1]+vertCount, face.Vertex[2]+vertCount, face.Vertex[2]+vertCount, face.Vertex[2]+vertCount)))
				}
			} else if hasvn {
				for _, face := range g.Faces {
					fl.Write([]byte(fmt.Sprintf(faceTemp12, face.Vertex[0]+vertCount, face.Vertex[0]+vertCount, face.Vertex[1]+vertCount, face.Vertex[1]+vertCount, face.Vertex[2]+vertCount, face.Vertex[2]+vertCount)))
				}
			} else if hasvt {
				for _, face := range g.Faces {
					fl.Write([]byte(fmt.Sprintf(faceTemp21, face.Vertex[0]+vertCount, face.Vertex[0]+vertCount, face.Vertex[1]+vertCount, face.Vertex[1]+vertCount, face.Vertex[2]+vertCount, face.Vertex[2]+vertCount)))
				}
			} else {
				for _, face := range g.Faces {
					fl.Write([]byte(fmt.Sprintf(faceTemp1, face.Vertex[0]+vertCount, face.Vertex[1]+vertCount, face.Vertex[2]+vertCount)))
				}
			}
		}
		vertCount += uint32(len(nd.Vertices))
	}
	fl.Close()

	for idx, m := range ms.Materials {
		mtlTex.Write([]byte(fmt.Sprintf("newmtl material_%d \n", idx)))
		mtlTex.Write([]byte("Ka 0.200000 0.200000 0.200000\n"))
		mtlTex.Write([]byte("Tr 1.000000\n"))
		mtlTex.Write([]byte("Ks 1.000000 1.000000 1.000000\n"))
		mtlTex.Write([]byte("Ns 000 \n"))
		mtlTex.Write([]byte("illum 2\n"))

		var tex *Texture
		cl := [3]byte{255, 255, 255}
		switch mtl := m.(type) {
		case *TextureMaterial:
			tex = mtl.Texture
			cl = mtl.Color
		case *PhongMaterial:
			tex = mtl.Texture
			cl = mtl.Color
		case *LambertMaterial:
			tex = mtl.Texture
			cl = mtl.Color
		case *BaseMaterial:
			cl = mtl.Color
		}

		if tex != nil {
			bt, _ := DecompressImage(tex.Data)
			width := int(tex.Size[0])
			height := int(tex.Size[1])
			img := image.NewNRGBA(image.Rect(0, 0, width, height))
			for y := height - 1; y > -1; y-- {
				for x := 0; x < width; x++ {
					index := y*width*4 + x*4
					if tex.Format == TEXTURE_FORMAT_RGBA {
						img.Set(x, y, color.NRGBA{
							R: bt[index],
							G: bt[index+1],
							B: bt[index+2],
							A: bt[index+3],
						})
					} else if tex.Format == TEXTURE_FORMAT_RGB {
						img.Set(x, y, color.NRGBA{
							R: bt[index],
							G: bt[index+1],
							B: bt[index+2],
							A: 255,
						})
					}
				}
			}
			imgNmae := fmt.Sprintf("node_tex_0_%d.jpg", tex.Id)
			im, _ := os.Create(filepath.Join(dir, imgNmae))
			jpeg.Encode(im, img, &jpeg.Options{Quality: 95})
			im.Close()
			mtlTex.Write([]byte(fmt.Sprintf("map_Kd %s \n", imgNmae)))
		} else {
			// bm, ok := m.(*PhongMaterial)
			// if !ok {
			// }
			mtlTex.Write([]byte(fmt.Sprintf("Kd %f %f %f  \n", float32(cl[0])/255, float32(cl[1])/255, float32(cl[2])/255)))
		}
	}
}

func TestVec(t *testing.T) {
	world := &vec3.T{-2389250.4338499242, 4518270.200871248, 3802675.424745363}
	head := &vec3.T{4.771371435839683, -0.753607839345932, 3.867249683942646}
	p := &vec3.T{4.802855, -0.753608, 3.828406}
	fmt.Println(p.Add(world).Length())
	world.Add(head)
	x, y, z, _ := proj.Ecef2Lonlat(p[0], p[1], p[2])
	fmt.Println(x, y, z)
}

func TestPipe(t *testing.T) {
	pos := []*fvec3.T{
		{-45.6055285647, 197.900406907, 631.169545605},
		{-55.3296683, 217.775199322, 601.643433287},
		{-57.99762254, 223.04394682, 593.7597909383},
	}
	lines := []string{"/home/hj/workspace/GISCore/build/temp/mst/yanshi/ys_zq_mdb/line/1.mst", "/home/hj/workspace/GISCore/build/temp/mst/yanshi/ys_zq_mdb/line/2.mst", "/home/hj/workspace/GISCore/build/temp/mst/yanshi/ys_zq_mdb/line/3.mst"}
	lines2 := []string{"tests/0.mst", "tests/1.mst", "tests/2.mst"}
	for i := 0; i < 3; i++ {
		ms, _ := MeshReadFrom(lines[i])
		for _, nd := range ms.Nodes {
			for k := range nd.Vertices {
				nd.Vertices[k].Add(pos[i])
			}
		}
		MeshWriteTo(lines2[i], ms)
		MstToObj(lines2[i], fmt.Sprintf("%d", i))
	}
}

func TestMst2Gltf(t *testing.T) {
	f, _ := os.Open("./tests/test1.mst")
	mh := MeshUnMarshal(f)
	doc := CreateDoc()
	BuildGltf(doc, mh, false, true)
	bt, _ := GetGltfBinary(doc, 8)
	ioutil.WriteFile("tests/test1.glb", bt, os.ModePerm)
}
