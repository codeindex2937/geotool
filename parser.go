package geotool

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/codeindex2937/geotool/shp"
	"github.com/codeindex2937/zipper"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
)

type CentroidPoint struct {
	*geojson.Feature
}

func (cp CentroidPoint) Point() orb.Point {
	// this is where you would decide how to define
	// the representative point of the feature.
	c, _ := planar.CentroidArea(cp.Geometry)
	return c
}

func ReadZip(source string, translator func(x, y float64) (float64, float64)) ([]*geojson.Feature, error) {
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

	records, err := ReadDbf(bytes.NewReader(dbfBuf.Bytes()))
	if err != nil {
		return nil, err
	}

	geometries := ReadShp(bytes.NewReader(shpBuf.Bytes()), int64(shpBuf.Len()), translator)

	featuries := []*geojson.Feature{}
	for i, geometry := range geometries {
		f := geojson.NewFeature(geometry)
		f.Properties = records[i]
		featuries = append(featuries, f)
	}

	return featuries, nil
}

func ReadShp(r io.ReadSeeker, size int64, translator func(x, y float64) (float64, float64)) []orb.Geometry {
	shpReader := shp.NewReader(r, size)

	geometries := []orb.Geometry{}

	for shpReader.Next() {
		_, p := shpReader.Shape()

		poly := p.(*shp.Polygon)
		polygon := orb.Polygon{}
		for i, offset := range poly.Parts {
			var end int32
			if i == int(poly.NumParts)-1 {
				end = poly.NumPoints
			} else {
				end = poly.Parts[i+1]
			}

			ring := orb.Ring{}
			for _, p := range poly.Points[offset:end] {
				x, y := translator(p.X, p.Y)
				ring = append(ring, orb.Point{x, y})
			}

			polygon = append(polygon, ring)
		}
		geometries = append(geometries, orb.MultiPolygon{polygon})
	}

	return geometries
}

func ReadDbf(src io.ReadSeeker) ([]map[string]interface{}, error) {
	dbfReader, err := shp.NewDbfReader(src)
	if err != nil {
		return nil, err
	}

	infos := []map[string]interface{}{}
	for i := 0; i < dbfReader.Length; i++ {
		entry, err := dbfReader.Read(i)
		if err != nil {
			return nil, err
		}

		infos = append(infos, entry)
	}

	return infos, nil
}
