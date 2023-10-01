package shp

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// ShpReader provides a interface for reading Shapefiles. Calls
// to the Next method will iterate through the objects in the
// Shapefile. After a call to Next the object will be available
// through the Shape method.
type ShpReader struct {
	GeometryType ShapeType
	bbox         Box
	err          error

	shp   io.ReadSeeker
	shape Shape
	num   int32
	size  int64
}

func NewReader(r io.ReadSeeker, size int64) *ShpReader {
	s := &ShpReader{shp: r, size: size}
	s.readHeaders()

	return s
}

// BBox returns the bounding box of the shapefile.
func (r *ShpReader) BBox() Box {
	return r.bbox
}

// Read and parse headers in the Shapefile. This will
// fill out GeometryType, filelength and bbox.
func (r *ShpReader) readHeaders() {
	// don't trust the the filelength in the header
	r.size, _ = r.shp.Seek(0, io.SeekEnd)

	var filelength int32
	r.shp.Seek(24, 0)
	// file length
	binary.Read(r.shp, binary.BigEndian, &filelength)
	r.shp.Seek(32, 0)
	binary.Read(r.shp, binary.LittleEndian, &r.GeometryType)
	r.bbox.MinX = readFloat64(r.shp)
	r.bbox.MinY = readFloat64(r.shp)
	r.bbox.MaxX = readFloat64(r.shp)
	r.bbox.MaxY = readFloat64(r.shp)
	r.shp.Seek(100, 0)
}

func readFloat64(r io.Reader) float64 {
	var bits uint64
	binary.Read(r, binary.LittleEndian, &bits)
	return math.Float64frombits(bits)
}

// Shape returns the most recent feature that was read by
// a call to Next. It returns two values, the int is the
// object index starting from zero in the shapefile which
// can be used as row in ReadAttribute, and the Shape is the object.
func (r *ShpReader) Shape() (int, Shape) {
	return int(r.num) - 1, r.shape
}

// newShape creates a new shape with a given type.
func newShape(shapetype ShapeType) (Shape, error) {
	switch shapetype {
	case NULL:
		return new(Null), nil
	case POINT:
		return new(Point), nil
	case POLYLINE:
		return new(PolyLine), nil
	case POLYGON:
		return new(Polygon), nil
	case MULTIPOINT:
		return new(MultiPoint), nil
	case POINTZ:
		return new(PointZ), nil
	case POLYLINEZ:
		return new(PolyLineZ), nil
	case POLYGONZ:
		return new(PolygonZ), nil
	case MULTIPOINTZ:
		return new(MultiPointZ), nil
	case POINTM:
		return new(PointM), nil
	case POLYLINEM:
		return new(PolyLineM), nil
	case POLYGONM:
		return new(PolygonM), nil
	case MULTIPOINTM:
		return new(MultiPointM), nil
	case MULTIPATCH:
		return new(MultiPatch), nil
	default:
		return nil, fmt.Errorf("Unsupported shape type: %v", shapetype)
	}
}

// Next reads in the next Shape in the Shapefile, which
// will then be available through the Shape method. It
// returns false when the reader has reached the end of the
// file or encounters an error.
func (r *ShpReader) Next() bool {
	cur, _ := r.shp.Seek(0, io.SeekCurrent)
	if cur >= r.size {
		return false
	}

	var size int32
	var shapetype ShapeType
	er := &errReader{Reader: r.shp}
	binary.Read(er, binary.BigEndian, &r.num)
	binary.Read(er, binary.BigEndian, &size)
	binary.Read(er, binary.LittleEndian, &shapetype)
	if er.e != nil {
		if er.e != io.EOF {
			r.err = fmt.Errorf("Error when reading metadata of next shape: %v", er.e)
		} else {
			r.err = io.EOF
		}
		return false
	}

	var err error
	r.shape, err = newShape(shapetype)
	if err != nil {
		r.err = fmt.Errorf("Error decoding shape type: %v", err)
		return false
	}
	r.shape.read(er)
	if er.e != nil {
		r.err = fmt.Errorf("Error while reading next shape: %v", er.e)
		return false
	}

	// move to next object
	r.shp.Seek(int64(size)*2+cur+8, 0)
	return true
}

// Err returns the last non-EOF error encountered.
func (r *ShpReader) Err() error {
	if r.err == io.EOF {
		return nil
	}
	return r.err
}
