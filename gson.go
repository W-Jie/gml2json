package main

type Gson struct {
	Type     string     `json:"type"`
	Crs      Crs        `json:"crs"`
	Features []Features `json:"features"`
}

type Crs struct {
	Type       string        `json:"type"`
	Properties CrsProperties `json:"properties"`
}

type CrsProperties struct {
	Name string `json:"name"`
}

type Features struct {
	Type       string     `json:"type"`
	Properties Properties `json:"properties"`
	Geometry   Geometry   `json:"geometry"`
}

type Properties struct {
	SmUserID   uint32  `json:"smuserid"`   //格式转换生成的id
	Area       float64 `json:"area"`       //区域面积
	RefName    string  `json:"refname"`    //小区、中区：对应原表中的REFNAME字段；大区留空
	Name       string  `json:"name"`       //区域名称
	RefID      uint32  `json:"refid"`      //上级区域编号:区域所属上级区域ID。其中小区的上级为中区；中区的上级为大区
	DistrictID uint32  `json:"districtid"` //行政区编号:区域所属行政区ID
	Ver        string  `json:"ver"`        //区域版本,示例：v1.20170704
	Class      string  `json:"class"`      //区域类型
	Id         uint32  `json:"id"`         //区域编号:唯一编号
	VerDate    string  `json:"verdate"`    //版本日期,示例：20170704
	OriID      uint32  `json:"oriid"`      //交研所数据ID
	//Center     Center  `json:"center"`     //中心点坐标
}

// type Center struct {
// 	Lon float32 `json:"lon"` // 中心点坐标，经度
// 	Lat float32 `json:"lat"` // 中心点坐标，纬度
// }

type Geometry struct {
	Type        string      `json:"type"`        //类型
	Coordinates interface{} `json:"coordinates"` //坐标点
}
