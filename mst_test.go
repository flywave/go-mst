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
)

const absPath = "/home/hj/workspace/GISCore/build/public/Resources/"

func TestToMst(t *testing.T) {
	dirs := []string{"model"} //"anchormodel",
	for _, d := range dirs {
		dr := absPath + d
		fs, _ := readDir(dr, dr, []string{".json"})
		for _, f := range fs {
			fpath := dr + f
			mstPh := strings.Replace(fpath, ".json", ".mst", 1)
			glbPh := strings.Replace(mstPh, ".mst", ".glb", 1)

			ThreejsBin2Mst(fpath)
			f, _ := os.Open(mstPh)
			mh := MeshUnMarshal(f)
			doc := CreateDoc()
			BuildGltf(doc, mh)
			bt, _ := GetGltfBinary(doc, 8)
			ioutil.WriteFile(glbPh, bt, os.ModePerm)
		}
	}
}

func TestGltf(t *testing.T) {
	f, _ := os.Open("/home/hj/workspace/GISCore/build/public/Resources/anchormodel/public/5025tiaoyaqi/5025tiyaqi.mst")
	mh := MeshUnMarshal(f)
	doc := CreateDoc()
	BuildGltf(doc, mh)
	bt, _ := GetGltfBinary(doc, 8)
	ioutil.WriteFile("tests/5025tiyaqi.gltf", bt, os.ModePerm)
}

func TestBin(t *testing.T) {
	ThreejsBin2Mst("/home/hj/workspace/GISCore/build/public/Resources/model/zbrl/relijg/JingGai_RL.json")
	MstToObj("/home/hj/workspace/GISCore/build/public/Resources/model/zbrl/relijg/JingGai_RL.mst", "JingGai_RL")

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
