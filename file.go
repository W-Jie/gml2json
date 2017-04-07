package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func readfile(file string) []byte {
	ifile, err := ioutil.ReadFile(file)

	//fix xml unmarshal error:  encoding "GB2312" declared but Decoder.CharsetReader is nil
	content := strings.Replace(string(ifile), "GB2312", "UTF-8", 1)

	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("成功读取文件 [%s]\n", file)

	return []byte(content)
}

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
