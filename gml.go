package main

type Gml struct {
	FeatureMember []FeatureMember `xml:"FeatureCollection>featureMember>TAZR"`
}

type FeatureMember struct {
	SmUserID         uint16  `xml:"SmUserID"`
	Id               float32 `xml:"ID"`
	Area             float32 `xml:"AREA"`
	Refname          string  `xml:"REFNAME"`
	Node             uint16  `xml:"NODE"`
	Tag              uint16  `xml:"TAG"`
	GeometryProperty string  `xml:"geometryProperty>Polygon>exterior>LinearRing>posList"`
}
