## gml2json ##
##### convert gml to json #####

#### Install ####
```shell
cd go2json
go build
```

#### Usage ####
```shell

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

```
