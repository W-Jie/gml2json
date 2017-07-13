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
