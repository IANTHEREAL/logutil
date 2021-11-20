package main

import (
	"go/build"
	"log"

	extractor "github.com/IANTHEREAL/logutil/extractor/go"
)

func main() {
	c := extractor.NewPackageComplier(build.Default, "github.com/IANTHEREAL/logutil/extractor/go")
	compliant, err := c.Compile()
	if err != nil {
		log.Fatalf("package compilation failed %s", err)
	}
	log.Printf("package compliant %s %+v", compliant.ImportPath, compliant.Repo)

	/*	for name, fi := range compliant.SourceFiles {
			log.Printf("source file %s %+v", name, fi)
		}

		for name, fi := range compliant.Deps {
			log.Printf("deps  %s %s %+v %+v", name, fi.ImportPath, fi.Repo, fi.SourceFiles[name])
		}*/
}
