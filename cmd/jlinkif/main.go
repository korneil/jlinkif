package main

import (
	"fmt"
	"github.com/korneil/jlinkif/internal"
	"github.com/mingrammer/cfmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

func main() {
	cfg := &internal.Config{}

	f, err := ioutil.ReadFile("jlinkif.yml")
	if err != nil {
		cfmt.Errorln("Error reading jlinkif.yml")
		os.Exit(0)
	}
	if err = yaml.Unmarshal(f, cfg); err != nil {
		cfmt.Errorln("Error parsing jlinkif.yml")
		os.Exit(0)
	}

	ctx, err := internal.NewContext(cfg)
	if err != nil {
		cfmt.Errorln(err)
	}
	defer ctx.Close()

	cfmt.Successln("Running with config:")
	fmt.Print(string(ctx.GetConfigYAML()))

	if err := ctx.Run(); err != nil {
		cfmt.Errorln(err)
	}
	ctx.Wait()
}
