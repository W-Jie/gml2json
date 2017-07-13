package main

import (
	"log"
	"os"
)

func savefile(file string) {
	// save to file
	ofile, err := os.OpenFile(file, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer ofile.Close()

	ofile.WriteString(string(gson))
	log.Printf("保存成功 [%s] \n", file)
}
