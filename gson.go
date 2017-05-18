package main

import "time"

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
	Id      uint16  `json:"ID"`
	Area    float32 `json:"AREA"`
	Refname string  `json:"REFNAME"`
	Node    uint16  `json:"NODE"`
	Tag     uint16  `json:"TAG"`
}

type Geometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

type Record struct {
	Smuserid uint16    `json:"smuserid"`
	Id       uint16    `json:"id"`
	Area     float32   `json:"area"`
	Refname  string    `json:"refname"`
	Node     uint16    `json:"node"`
	Tag      uint16    `json:"tag"`
	Geometry Geometry  `json:"geometry"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}
