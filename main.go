package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	infile  = flag.String("i", "example.gml", "")
	outfile = flag.String("o", "convert.json", "")
)

var usage = `
Usage: gml2json [options...]

Options:
  -i  Input gml file. (default "example.gml")
  -o  Output json type. (default "convert.json")

Example:

  gml2json -i example.gml -o example.json 
`

var gson []byte

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()

	ifile, err := ioutil.ReadFile(*infile)

	//fix xml unmarshal error:  encoding "GB2312" declared but Decoder.CharsetReader is nil
	content := strings.Replace(string(ifile), "GB2312", "UTF-8", 1)

	if err != nil {
		log.Fatalln(err)
		return
	}
	log.Printf("read file [%s] successfully!\n", *infile)

	var gml Gml

	err = xml.Unmarshal([]byte(content), &gml)
	if err != nil {
		log.Fatalln(err)
		return
	}

	features := []Features{}

	for _, fMenber := range gml.FeatureMember {

		coordinates := make([]interface{}, 0)
		coordinate := make([]interface{}, 0)

		for _, line := range strings.Split(strings.TrimSpace(fMenber.GeometryProperty), "\n") {
			line = strings.Replace(strings.TrimSpace(line), "\t", "", -1)
			point := make([]float64, 0)
			for _, value := range strings.Split(line, " ") {
				p, _ := strconv.ParseFloat(value, 64)
				point = append(point, p)
			}
			coordinate = append(coordinate, point)

		}
		coordinates = append(coordinates, coordinate)

		ftr := Features{
			Type: "Feature",
			Properties: Properties{
				Id:      int(fMenber.Id),
				Area:    fMenber.Area,
				Refname: fMenber.Refname,
				Node:    fMenber.Node,
				Tag:     fMenber.Tag,
			},
			Geometry: Geometry{
				Type:        "Polygon",
				Coordinates: coordinates,
			},
		}
		features = append(features, ftr)
	}

	j := Gson{
		Type: "FeatureCollection",
		Crs: Crs{
			Type: "name",
			Properties: CrsProperties{
				Name: "urn:ogc:def:crs:OGC:1.3:CRS84",
			},
		},
		Features: features,
	}

	gson, err = json.Marshal(&j)
	if err != nil {
		log.Fatalln(err)
		return
	}

	// save to file
	ofile, err := os.OpenFile(*outfile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer ofile.Close()

	ofile.WriteString(string(gson))
	log.Printf("save to file [%s] successfully!\n", *outfile)
}
