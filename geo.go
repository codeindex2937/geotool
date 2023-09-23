package geotool

import (
	"math"
)

var a = 6378.137          // km
var f = 1 / 298.257222101 // GRS80
var lon0 = 121.0
var N0 = 0.0   // km
var E0 = 250.0 // km
var k0 = 0.9999
var n = f / (2 - f)
var A = a / (1 + n) * (1 + math.Pow(n, 2)/4 + math.Pow(n, 4)/64)
var b1 = n/2 - math.Pow(n, 2)*2/3 + math.Pow(n, 3)*37/96
var b2 = math.Pow(n, 2)/48 + math.Pow(n, 3)/15
var b3 = math.Pow(n, 3) * 17 / 480
var d1 = 2*n - math.Pow(n, 2)*2/3 - 2*math.Pow(n, 3)
var d2 = math.Pow(n, 2)*7/3 - math.Pow(n, 3)*8/5
var d3 = math.Pow(n, 3) * 56 / 15

type TWD97 struct {
	X float64
	Y float64
}

type WGS84 struct {
	X float64
	Y float64
}

func ToWGS(p TWD97) WGS84 {
	x := p.X / 1000.0
	y := p.Y / 1000.0

	xi := (y - N0) / (k0 * A)
	eta := (x - E0) / (k0 * A)
	xip := (xi -
		b1*math.Sin(2*xi)*math.Cosh(2*eta) -
		b2*math.Sin(4*xi)*math.Cosh(4*eta) -
		b3*math.Sin(6*xi)*math.Cosh(6*eta))
	etap := (eta -
		b1*math.Cos(2*xi)*math.Sinh(2*eta) -
		b2*math.Cos(4*xi)*math.Sinh(4*eta) -
		b3*math.Cos(6*xi)*math.Sinh(6*eta))
	chi := math.Asin(math.Sin(xip) / math.Cosh(etap))
	lat := degrees(
		chi +
			d1*math.Sin(2*chi) +
			d2*math.Sin(4*chi) +
			d3*math.Sin(6*chi))
	lng := lon0 + degrees(
		math.Atan2(math.Sinh(etap), math.Cos(xip)))

	return WGS84{X: lng, Y: lat}
}

func degrees(grad float64) float64 {
	return grad / math.Pi * 180
}
