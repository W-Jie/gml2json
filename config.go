package main

type Config struct {
	APPName string `default:"gml2json"`

	DB struct {
		User     string `required:"true"`
		Password string `required:"true"`
		Tnsname  string `required:"true"`
	}
}
