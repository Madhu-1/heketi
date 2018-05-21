package glusterd

import (
	"github.com/gluster/glusterd2/pkg/restclient"
	"github.com/heketi/heketi/executors/cmdexec"
	"github.com/heketi/heketi/pkg/utils"
)

type Gluster struct {
	Client *restclient.Client
	Config GlusterdConfig
	cmdexec.CmdExecutor
}

type GlusterdConfig struct {
	SCHEMA   string `json:"url_schema"`
	PORT     string `json:"port"`
	CertPath string `json:"cert_path"`
	Insecure bool   `json:"insecure"`
}

var (
	logger = utils.NewLogger("[glusterd]", utils.LEVEL_DEBUG)
)

func InitRESTClient(config *GlusterdConfig) (*Gluster, error) {
	g := Gluster{}
	//TODO add code read certfile and pass it
	g.Config = *config
	return &g, nil
}

func (g *Gluster) createClient(host string) {
	//add default ip if not present
	url := g.Config.SCHEMA + host + g.Config.PORT
	g.Client = restclient.New(url, "", "", g.Config.CertPath, g.Config.Insecure)
}
