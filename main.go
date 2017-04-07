package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/jinzhu/configor"
	_ "github.com/mattn/go-oci8"
)

var (
	count    = 0
	sqlcount = 0
	wg       = &sync.WaitGroup{}
	gson     []byte
)

var (
	infile     = flag.String("i", "example.gml", "")
	outfile    = flag.String("o", "convert.json", "")
	configfile = flag.String("c", "config.yaml", "")
)

var usage = `
Usage: gml2json [options...]

Options:
  -i  Input gml file. (default "example.gml")
  -o  Output json file. (default "convert.json")
  -c  Config file. (default "config.yaml")

Example:

  gml2json -i example.gml -o example.json -c config.yaml
`

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()

	Config := Config{}
	configor.Load(&Config, *configfile)

	log.Printf("starting %s ", Config.APPName)

	//os.Setenv("NLS_LANG", "")

	// 用户名/密码@实例名
	connect := Config.DB.User + "/" + Config.DB.Password + "@" + Config.DB.Tnsname
	db, err := sql.Open("oci8", connect)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	content := readfile(*infile)

	var gml Gml

	err = xml.Unmarshal(content, &gml)
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

		geometry := Geometry{
			Type:        "Polygon",
			Coordinates: coordinates,
		}

		ftr := Features{
			Type: "Feature",
			Properties: Properties{
				Id:      uint16(fMenber.Id),
				Area:    fMenber.Area,
				Refname: fMenber.Refname,
				Node:    fMenber.Node,
				Tag:     fMenber.Tag,
			},
			Geometry: geometry,
		}
		features = append(features, ftr)

		record, err := json.Marshal(&geometry)
		if err != nil {
			log.Fatalln(err)
			return
		}
		count += 1

		wg.Add(1)
		go insertdb(db, fMenber.SmUserID, uint16(fMenber.Id), fMenber.Area, fMenber.Refname, fMenber.Node, fMenber.Tag, string(record))

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

	savefile(*outfile)
	wg.Wait()
	log.Printf("转换已完成！解析数据量: %d ，成功入库量: %d ", count, sqlcount)
}

func insertdb(db *sql.DB, smuserid uint16, id uint16, area float32, refname string, node uint16, tag uint16, geometry string) {
	_, err := db.Exec("insert into geojson(smuserid,id, area, refname, node, tag, geometry) values(:1,:2,:3,:4,:5,:6,:7)", smuserid, id, area, refname, node, tag, geometry)
	if err != nil {
		log.Println(err)
	} else {
		sqlcount += 1
	}
	wg.Done()
}
