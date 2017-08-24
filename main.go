package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beevik/etree"
	"github.com/garyburd/redigo/redis"
	"github.com/jinzhu/configor"
	_ "github.com/mattn/go-oci8"
	"github.com/tidwall/sjson"
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
  -ver  Geojson version.Format: YYYY-MM-DD,Exmple：20170101.(default: "20170101")
  -db   Enable save to database. (default: false)

Example:

  gml2json -i example.gml -o example.json -c config.yaml -t max -ver 20170101
`

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
		MaxIdle: config.Tile38.MaxIdle, // 最大的空闲连接数
		//MaxActive:   config.Tile38.MaxActive,                 // 最大的激活连接数
		IdleTimeout: config.Tile38.IdleTimeout * time.Second, // 最大的空闲连接等待时间
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(config.Tile38.Network, config.Tile38.Address)
			if err != nil {
				panic(err.Error())
			}
			//c.Do("SELECT", config.Tile38.Database)
			//c.Do("OUTPUT", "JSON") //设置输出格式为json
			return c, nil
		}, // 建立连接
		TestOnBorrow: func(c redis.Conn, t time.Time) error { // 测试链接可用性
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
	r := redisclient.Get()                // 获得连接
	ok, err := redis.String(r.Do("PING")) // 选择数据库
	if err != nil {
		log.Fatal(err)
	} else if ok == "PONG" {
		log.Printf("成功连接Redis：%v://%v", config.Tile38.Network, config.Tile38.Address)
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

	doc := etree.NewDocument()
	doc.ReadSettings.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}
	doc.ReadSettings.Permissive = true

	if err := doc.ReadFromFile(*infile); err != nil {
		panic(err)
	}

	features := []Features{}
	for _, featureMember := range doc.FindElements("//featureMember") {

		var smuserid, refid, districtid, id, oriid uint32
		var area float64
		var refname, name, ver, class, verdate, geometryType string

		if SmUserID := featureMember.FindElement("*/SmUserID"); SmUserID != nil {
			tmp, _ := strconv.Atoi(SmUserID.Text())
			smuserid = uint32(tmp)
		}
		if AREA := featureMember.FindElement("*/AREA"); AREA != nil {
			tmp, _ := strconv.ParseFloat(AREA.Text(), 64)
			area = tmp
		}
		if REFNAME := featureMember.FindElement("*/REFNAME"); REFNAME != nil {
			refname = REFNAME.Text()
		}
		if NAME := featureMember.FindElement("*/NAME"); NAME != nil {
			name = NAME.Text()
		}
		if REFID := featureMember.FindElement("*/REFID"); REFID != nil {
			tmp, _ := strconv.Atoi(REFID.Text())
			refid = uint32(tmp)
		}
		if DISTRICTID := featureMember.FindElement("*/DISTRICTID"); DISTRICTID != nil {
			tmp, _ := strconv.Atoi(DISTRICTID.Text())
			districtid = uint32(tmp)
		}
		if VER := featureMember.FindElement("*/VER"); VER != nil {
			ver = VER.Text()
		}
		if CLASS := featureMember.FindElement("*/CLASS"); CLASS != nil {
			class = CLASS.Text()
		}
		if ID := featureMember.FindElement("*/ID"); ID != nil {
			tmp, _ := strconv.Atoi(ID.Text())
			id = uint32(tmp)
		}
		if VERDATE := featureMember.FindElement("*/VERDATE"); VERDATE != nil {
			verdate = VERDATE.Text()
		}

		if oriID := featureMember.FindElement("*/oriID"); oriID != nil {
			tmp, _ := strconv.Atoi(oriID.Text())
			oriid = uint32(tmp)

		}

		if gType := featureMember.FindElement("*/geometryProperty"); gType != nil {
			geometryType = gType.ChildElements()[0].Tag
		}

		coordinates := make([]interface{}, 0)
		if geometryType == "MultiSurface" {
			geometryType = "MultiPolygon"
			for _, surfaceMember := range featureMember.FindElements("//surfaceMember") {
				multCoordinates := make([]interface{}, 0)
				for _, posList := range surfaceMember.FindElements("//posList") {
					coordinate := make([]interface{}, 0)
					for _, line := range strings.Split(strings.TrimSpace(posList.Text()), "\n") {
						line = strings.Replace(strings.TrimSpace(line), "\t", "", -1)
						point := make([]float64, 0)
						for _, value := range strings.Split(line, " ") {
							p, _ := strconv.ParseFloat(value, 64)
							point = append(point, p)
						}
						coordinate = append(coordinate, point)
					}
					multCoordinates = append(multCoordinates, coordinate)
				}
				coordinates = append(coordinates, multCoordinates)
			}
		} else {
			for _, posList := range featureMember.FindElements("//posList") {
				coordinate := make([]interface{}, 0)
				for _, line := range strings.Split(strings.TrimSpace(posList.Text()), "\n") {
					line = strings.Replace(strings.TrimSpace(line), "\t", "", -1)
					point := make([]float64, 0)
					for _, value := range strings.Split(line, " ") {
						p, _ := strconv.ParseFloat(value, 64)
						point = append(point, p)
					}
					coordinate = append(coordinate, point)
				}
				coordinates = append(coordinates, coordinate)
			}
		}

		count += 1

		ftr := Features{
			Type: "Feature",
			Properties: Properties{
				SmUserID:   uint32(smuserid),
				Area:       area,
				RefName:    refname,
				Name:       name,
				RefID:      uint32(refid),
				DistrictID: uint32(districtid),
				Ver:        ver,
				Class:      class,
				Id:         uint32(id),
				VerDate:    verdate,
				OriID:      uint32(oriid),
			},
			Geometry: Geometry{
				Type:        geometryType,
				Coordinates: coordinates,
			},
		}
		features = append(features, ftr)

		// 保存到redis
		wg.Add(1)
		go insertredis(redisclient, haskkey, ftr)

		// 保存到database
		if *save2db {
			wg.Add(1)
			go insertdb(db, ftr)
		}
	}

	//geojson 保存到文件
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

	savefile(*outfile, string(gson))

	wg.Wait()
	log.Printf("转换已完成！解析数据量: %d ，成功入库量(启用:%v): %d ,Redis新建/更新量：%d/%d ", count, *save2db, sqlcount, redisCreateCnt, redisUpdateCnt)
}

func insertdb(db *sql.DB, fm Features) {
	geometry, err := json.Marshal(fm.Geometry)
	if err != nil {
		log.Fatalln(err)
		return
	}
	_, err = db.Exec("insert into geojson(smuserid, id, area, name, refid, refname, districtid, class, ver, verdate, oriid, geometry) values(:1,:2,:3,:4,:5,:6,:7,:8,:9,:10,:11,:12)", fm.Properties.SmUserID, fm.Properties.Id, fm.Properties.Area, fm.Properties.Name, fm.Properties.RefID, fm.Properties.RefName, fm.Properties.DistrictID, fm.Properties.Class, fm.Properties.Ver, fm.Properties.VerDate, fm.Properties.OriID, geometry)
	if err != nil {
		log.Println(err)
	} else {
		sqlcount += 1
	}
	wg.Done()
}

func insertredis(redisclient *redis.Pool, haskkey string, fm Features) {

	hashvalue, err := json.Marshal(&fm)
	if err != nil {
		log.Println(err)
	}

	hashvalue, _ = sjson.SetBytes(hashvalue, "properties.center.lon", 0) // 中心点坐标，经度
	hashvalue, _ = sjson.SetBytes(hashvalue, "properties.center.lat", 0) // 中心点坐标，经度
	//t := time.Now()
	// hashvalue, _ = sjson.SetBytes(hashvalue, "properties.created", t)   // 创建时间，示例：2017-07-11T09:42:30.6541063+08:00
	// hashvalue, _ = sjson.SetBytes(hashvalue, "properties.updated", t)   // //更新时间，示例：2017-07-11T09:42:30.6541063+08:00

	r := redisclient.Get()
	created, err := r.Do("SET", haskkey, fm.Properties.Id, "OBJECT", string(hashvalue))
	if err != nil {
		log.Printf("%v:,haskkey:%v, fm.Properties.Id:%v, hashvalue:%v\n", err, haskkey, fm.Properties.Id, string(hashvalue))
	} else {
		mu.Lock()
		if created == "OK" {
			redisCreateCnt += 1
		} else {
			redisUpdateCnt += 1
		}
		mu.Unlock()
	}
	defer r.Close()
	defer wg.Done()
}
