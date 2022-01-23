package internal

import (
	"github.com/korneil/jlinkif/internal/watcher"
)

type GDBConfig struct {
	Run  bool     `yaml:"run"`
	Exec string   `yaml:"exec"`
	Args []string `yaml:"args"`
}

type RTTConfig struct {
	Address string `yaml:"address"`
}

type Config struct {
	Load       string         `yaml:"load"`
	SymbolFile string         `yaml:"symbol_file"`
	Root       string         `yaml:"root"`
	Watch      watcher.Config `yaml:"watch"`
	RTT        RTTConfig      `yaml:"rtt"`
	GDB        GDBConfig      `yaml:"gdb"`
}
