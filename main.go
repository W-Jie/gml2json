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
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/jinzhu/configor"
	_ "github.com/mattn/go-oci8"
)

var (
	count          = 0
	sqlcount       = 0
	redisCreateCnt = 0
	redisUpdateCnt = 0
	wg             = &sync.WaitGroup{}
	mu             sync.Mutex
	gson           []byte
)

var (
	infile      = flag.String("i", "example.gml", "")
	outfile     = flag.String("o", "convert.json", "")
	configfile  = flag.String("c", "config.yaml", "")
	hasktype    = flag.String("t", "max", "")
	haskversion = flag.String("ver", "20170101", "")
	save2db     = flag.Bool("db", false, "")
)

var usage = `
Usage: gml2json [options...]

Options:
  -i    Input gml file. (default: "example.gml")
  -o    Output json file. (default: "convert.json")
  -c    Config file. (default: "config.yaml")
  -t    Geojson type, max|mid|min. (default: "max")
  -ver  Geojson version.Format: YYYY-MM-DD,Exmple：20170101。(default: "20170101")
  -db   Enable save to database. (default: false)

Example:

  gml2json -i example.gml -o example.json -c config.yaml -t max -ver 20170101
`

// type Record struct {
// 	Smuserid uint64    `json:"smuserid"`
// 	Id       uint16    `json:"id"`
// 	Area     float32   `json:"area"`
// 	Refname  string    `json:"refname"`
// 	Node     uint16    `json:"node"`
// 	Tag      uint16    `json:"tag"`
// 	Geometry string    `json:"geometry"`
// 	Created  time.Time `json:"created"`
// 	Updated  time.Time `json:"updated"`
// }

var redisclient *redis.Pool

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()

	config := Config{}
	err := configor.Load(&config, *configfile)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("成功加载配置文件:%v ", *configfile)
	}

	log.Printf("开始运行 %s ", config.APPName)

	haskkey := *hasktype + "/v:" + *haskversion

	redisclient = &redis.Pool{
		MaxIdle:     config.Redis.MaxIdle,                   // 最大的空闲连接数
		MaxActive:   config.Redis.MaxActive,                 // 最大的激活连接数
		IdleTimeout: config.Redis.IdleTimeout * time.Second, // 最大的空闲连接等待时间
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(config.Redis.Network, config.Redis.Address)
			if err != nil {
				panic(err.Error())
			}
			c.Do("SELECT", config.Redis.Database)
			return c, nil
		}, // 建立连接
	}
	r := redisclient.Get()                // 获得连接
	ok, err := redis.String(r.Do("PING")) // 选择数据库
	if err != nil {
		log.Fatal(err)
	} else if ok == "PONG" {
		log.Printf("成功连接Redis：%v://%v, DB:%d", config.Redis.Network, config.Redis.Address, config.Redis.Database)
	}

	defer r.Close()

	//os.Setenv("NLS_LANG", "")

	// 用户名/密码@实例名
	connect := config.DB.User + "/" + config.DB.Password + "@" + config.DB.Tnsname
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
		go insertredis(redisclient, haskkey, fMenber.SmUserID, uint16(fMenber.Id), fMenber.Area, fMenber.Refname, fMenber.Node, fMenber.Tag, coordinates)

		if *save2db {
			wg.Add(1)
			go insertdb(db, fMenber.SmUserID, uint16(fMenber.Id), fMenber.Area, fMenber.Refname, fMenber.Node, fMenber.Tag, string(record))
		}
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
	log.Printf("转换已完成！解析数据量: %d ，成功入库量(启用:%v): %d ,Redis新建/更新量：%d/%d ", count, *save2db, sqlcount, redisCreateCnt, redisUpdateCnt)
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

func insertredis(redisclient *redis.Pool, haskkey string, smuserid uint16, id uint16, area float32, refname string, node uint16, tag uint16, coordinates []interface{}) {
	record := &Record{
		Smuserid: smuserid,
		Id:       id,
		Area:     area,
		Refname:  refname,
		Node:     node,
		Tag:      tag,
		Geometry: Geometry{
			Type:        "Polygon",
			Coordinates: coordinates,
		},
		Created: time.Now(),
		Updated: time.Now(),
	}

	hashvalue, err := json.Marshal(&record)
	if err != nil {
		log.Println(err)
	}

	r := redisclient.Get()
	created, err := redis.Bool(r.Do("HSET", haskkey, smuserid, (hashvalue)))
	if err != nil {
		log.Println(err)
	} else {
		mu.Lock()
		if created {
			redisCreateCnt += 1
		} else {
			redisUpdateCnt += 1
		}
		mu.Unlock()
	}
	defer r.Close()
	defer wg.Done()
}
