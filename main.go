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
			if geometryType == "MultiSurface" {
				multCoordinates := make([]interface{}, 0)
				multCoordinates = append(multCoordinates, coordinate)
				coordinates = append(coordinates, multCoordinates)
			} else {
				coordinates = append(coordinates, coordinate)
			}
		}

		fMember := FeatureMember{
			SmUserID: uint32(smuserid),
			Properties: Properties{
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
			Created: time.Now(),
			Updated: time.Now(),
		}

		count += 1

		wg.Add(1)
		go insertredis(redisclient, haskkey, fMember)

		ftr := Features{
			Type:       "Feature",
			Properties: fMember.Properties,
			Geometry: Geometry{
				Type:        geometryType,
				Coordinates: coordinates,
			},
		}

		features = append(features, ftr)

		//geojson 保存到数据库
		record, err := json.Marshal(&ftr.Geometry)
		if err != nil {
			log.Fatalln(err)
			return
		}

		if *save2db {
			wg.Add(1)
			go insertdb(db, fMember, string(record))
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

	savefile(*outfile)

	wg.Wait()
	log.Printf("转换已完成！解析数据量: %d ，成功入库量(启用:%v): %d ,Redis新建/更新量：%d/%d ", count, *save2db, sqlcount, redisCreateCnt, redisUpdateCnt)
}

func insertdb(db *sql.DB, fm FeatureMember, geometry string) {
	_, err := db.Exec("insert into geojson(smuserid, id, area, name, refid, refname, districtid, class, ver, verdate, oriid, geometry) values(:1,:2,:3,:4,:5,:6,:7,:8,:9,:10,:11,:12)", fm.SmUserID, fm.Id, fm.Area, fm.Name, fm.RefID, fm.RefName, fm.DistrictID, fm.Class, fm.Ver, fm.VerDate, fm.OriID, geometry)
	if err != nil {
		log.Println(err)
	} else {
		sqlcount += 1
	}
	wg.Done()
}

func insertredis(redisclient *redis.Pool, haskkey string, fm FeatureMember) {

	hashvalue, err := json.Marshal(&fm)
	if err != nil {
		log.Println(err)
	}

	r := redisclient.Get()
	created, err := redis.Bool(r.Do("HSET", haskkey, fm.Id, hashvalue))
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
