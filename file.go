package main

import (
	"log"
	"os"
)

func savefile(file, context string) {
	// save to file
	ofile, err := os.OpenFile(file, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer ofile.Close()

	ofile.WriteString(context)
	log.Printf("保存成功 [%s] \n", file)
}
