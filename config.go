package main

import "time"

type Config struct {
	APPName string `default:"gml2json"`

	DB struct {
		User     string `default:"username" required:"true"`
		Password string `default:"password" required:"true"`
		Tnsname  string `default:"tnsname" required:"true"`
	}

	Redis struct {
		Network     string        `default:"tcp"  required:"true"`
		Address     string        `default:"127.0.0.1:6379"  required:"true"`
		MaxIdle     int           `default:1`
		IdleTimeout time.Duration `default:120`
		Database    int           `default:0`
	}
}
