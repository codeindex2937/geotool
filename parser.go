package geotool

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/codeindex2937/geotool/shp"
	"github.com/codeindex2937/zipper"
)

type Area struct {
	Points []WGS84
	Bound  shp.Box
}

type AreaInfo interface {
	SetAreas([]Area)
	GetAreas() []Area
	Apply(data map[string]interface{})
}

func ReadZip[T any, U interface {
	AreaInfo
	*T
}](source string, translator func(x, y float64) (float64, float64)) ([]U, error) {
	ext := filepath.Ext(source)
	filename := strings.TrimSuffix(source, ext)

	f, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	r, err := zipper.NewReader(f, stat.Size())
	if err != nil {
		return nil, err
	}

	dbfBuf := bytes.NewBuffer([]byte{})
	shpBuf := bytes.NewBuffer([]byte{})
	receivers := map[string]io.Writer{
		filename + ".dbf": dbfBuf,
		filename + ".shp": shpBuf,
	}

	if err := r.ReadFiles(receivers); err != nil {
		return nil, err
	}

	dbfInfos, err := ReadDbf[T, U](bytes.NewReader(dbfBuf.Bytes()))
	if err != nil {
		return nil, err
	}

	areaGroups := ReadShp[T](bytes.NewReader(shpBuf.Bytes()), int64(shpBuf.Len()), translator)

	for i := range dbfInfos {
		dbfInfos[i].SetAreas(areaGroups[i])
	}

	return dbfInfos, nil
}

func ReadShp[T any](r io.ReadSeeker, size int64, translator func(x, y float64) (float64, float64)) [][]Area {
	shpReader := shp.NewReader(r, size)
	areaGroups := [][]Area{}

	for shpReader.Next() {
		_, p := shpReader.Shape()

		poly := p.(*shp.Polygon)
		areas := []Area{}
		for i, offset := range poly.Parts {
			var end int32
			if i == int(poly.NumParts)-1 {
				end = poly.NumPoints
			} else {
				end = poly.Parts[i+1]
			}

			points := []WGS84{}
			for _, p := range poly.Points[offset:end] {
				x, y := translator(p.X, p.Y)
				points = append(points, WGS84{X: x, Y: y})
			}
			areas = append(areas, Area{Bound: poly.BBox(), Points: points})
		}

		areaGroups = append(areaGroups, areas)
	}

	return areaGroups
}

func ReadDbf[T any, U interface {
	AreaInfo
	*T
}](src io.ReadSeeker) ([]U, error) {
	dbfReader, err := shp.NewDbfReader(src)
	if err != nil {
		return nil, err
	}

	infos := []U{}
	for i := 0; i < dbfReader.Length; i++ {
		entry, err := dbfReader.Read(i)
		if err != nil {
			return nil, err
		}

		info := U(new(T))
		info.Apply(entry)

		infos = append(infos, info)
	}

	return infos, nil
}
