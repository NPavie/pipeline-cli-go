package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/kardianos/osext"
	"launchpad.net/goyaml"
)

// Yaml file keys
const (
	HOST         = "host"
	PORT         = "port"
	PATH         = "ws_path"
	APPPATH      = "app_path"
	CLIENTKEY    = "client_key"
	CLIENTSECRET = "client_secret"
	DEBUG        = "debug"
	STARTING     = "starting"
)

// Other convinience constants
const (
	ERR_STR      = "Error parsing configuration: %v"
	DEFAULT_FILE = "config.yml"
)

// Config is just a map
type Config map[string]interface{}

// Default minimal configuration
var config = Config{

	HOST:         "http://localhost",
	PORT:         8181,
	PATH:         "ws",
	APPPATH:      "DAISY Pipeline",
	CLIENTKEY:    "",
	CLIENTSECRET: "",
	DEBUG:        false,
	STARTING:     false,
}

// Config items descriptions
var config_descriptions = map[string]string{

	HOST:         "Pipeline's webservice host",
	PORT:         "Pipeline's webserivce port",
	PATH:         "Pipeline's webservice path, as in http://daisy.org:8181/path",
	APPPATH:      "DAISY Pipeline app executable path",
	CLIENTKEY:    "Client key for authenticated requests",
	CLIENTSECRET: "Client secrect for authenticated requests",
	DEBUG:        "Print debug messages. true or false. ",
	STARTING:     "Start the webservice in the local computer if it is not running. true or false",
}

// Makes a copy of the default config
func copyConf() Config {
	ret := make(Config)
	for k, v := range config {
		ret[k] = v
	}
	return ret
}

// Tries to load the default configuration file ( folder where the executable is located / config.yml) if not possible
// returns a minimal configuration setup
func NewConfig() Config {
	cnf := copyConf()
	if err := loadDefault(cnf); err != nil {
		fmt.Println("Warning : no default configuration file found")
		log.Println(err.Error())
		return copyConf()
	}
	return cnf
}

// Loads the default configuration file
func loadDefault(cnf Config) error {
	folder, err := osext.ExecutableFolder()
	if err != nil {
		return err
	}
	file, err := os.Open(folder + string(os.PathSeparator) + DEFAULT_FILE)
	if err != nil {
		return err
	}
	defer file.Close()
	err = cnf.FromYaml(file)
	if err != nil {
		return err
	}
	return nil
}

// Loads the contents of the yaml file into the configuration
func (c Config) FromYaml(r io.Reader) error {
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	err = goyaml.Unmarshal(bytes, c)
	if err != nil {
		return err
	}
	c.UpdateDebug()
	return err
}

// This method should be called if the DEBUG configuration is changed. The internal Config methods
// do this automatically
func (c Config) UpdateDebug() {
	if !c[DEBUG].(bool) {
		log.SetOutput(ioutil.Discard)
	} else {
		log.SetOutput(os.Stdout)
	}
}

// Returns the Url composed by HOSTNAME:PORT/PATH/
func (c Config) Url() string {
	return fmt.Sprintf("%v:%v/%v/", c[HOST], c[PORT], c[PATH])
}
func (c Config) ExecPath() string {
	base, err := osext.ExecutableFolder()
	if err != nil {
		panic("Error getting executable path")
	}
	return c.buildPath(base)
}

func (c Config) buildPath(base string) string {
	execpath := c[APPPATH].(string)
	// empty app path defaults to looking for DAISY Pipeline app in the PATH
	if execpath == "" {
		execpath = "DAISY Pipeline"
	}

	switch runtime.GOOS {
	case "windows":
		// on windows, check by its extension if the provided runner is a script
		// or an executable, and add the .exe extension if it's not there
		// (to handle default execpath value)
		winExt := []string{".exe", ".bat", ".cmd", ".ps1"}
		if !slices.Contains(winExt, execpath[len(execpath)-4:]) {
			execpath += ".exe"
		}
	}
	if path, _err := exec.LookPath(execpath); _err == nil {
		// exec found in path, return the absolute path of the app
		return filepath.FromSlash(path)
	}
	p := filepath.FromSlash(execpath)
	if filepath.IsAbs(p) {
		return p
	} else {
		return base + string(filepath.Separator) + p
	}
}
