package main

type Gml struct {
	FeatureMember []FeatureMember `xml:"FeatureCollection>featureMember>TAZR"`
}

type FeatureMember struct {
	SmUserID         int     `xml:"SmUserID"`
	Id               float64 `xml:"ID"`
	Area             float32 `xml:"AREA"`
	Refname          string  `xml:"REFNAME"`
	Node             int64   `xml:"NODE"`
	Tag              int     `xml:"TAG"`
	GeometryProperty string  `xml:"geometryProperty>Polygon>exterior>LinearRing>posList"`
}
