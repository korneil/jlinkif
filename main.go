package main

import (
	"embed"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/korneil/jlinkif/internal"
	"github.com/mingrammer/cfmt"
	"gopkg.in/yaml.v3"
)

//go:embed jlinkif.default.yml
var fs embed.FS

func main() {
	ctx := &internal.Context{}

	if defaultCfgBytes, err := fs.ReadFile("jlinkif.default.yml"); err != nil {
		cfmt.Errorln("Error reading embedded jlinkif.default.yml")
		os.Exit(1)
	} else if err = yaml.Unmarshal(defaultCfgBytes, &ctx.Config); err != nil {
		cfmt.Errorln("Error parsing jlinkif.default.yml")
		os.Exit(1)
	}

	f, err := ioutil.ReadFile("jlinkif.yml")
	if err != nil {
		cfmt.Errorln("Error reading jlinkif.yml")
		os.Exit(1)
	}
	if err = yaml.Unmarshal(f, &ctx.Config); err != nil {
		cfmt.Errorln("Error parsing jlinkif.yml")
		os.Exit(1)
	}

	if err := internal.Init(ctx); err != nil {
		cfmt.Errorln(err)
		os.Exit(1)
	}
	defer ctx.Close()

	cfmt.Successln("Running with config:")
	fmt.Print(string(ctx.GetConfigYAML()))

	if err := ctx.Run(); err != nil {
		cfmt.Errorln(err)
	}
	ctx.Wait()
}
