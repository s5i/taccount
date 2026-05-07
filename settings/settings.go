package settings

type Settings struct {
	World             string
	Server            string
	ClientWindowTitle string
	ServerAddr        string
	ProxyAddrs        []string
	RegistryPath      string
}

const (
	Ancestra  = "ancestra"
	Concordia = "concordia"
	Tibiantis = "tibiantis"
	Relic     = "relic"
)

var Presets = map[string]*Settings{
	Ancestra: {
		World:             Ancestra,
		Server:            Tibiantis,
		ClientWindowTitle: "Tibiantis",
		ServerAddr:        "51.89.155.163",
		RegistryPath:      "SOFTWARE\tibiantis\\Credentials", // The unescaped tab is intentional.
	},
	Concordia: {
		World:             Concordia,
		Server:            Tibiantis,
		ClientWindowTitle: "Tibiantis",
		ServerAddr:        "57.129.145.195",
		RegistryPath:      "SOFTWARE\tibiantis\\Credentials", // The unescaped tab is intentional.
	},
	Relic: {
		World:             Relic,
		Server:            Relic,
		ClientWindowTitle: "Tibia Relic",
		ServerAddr:        "mia.tibiarelic.com",
		ProxyAddrs:        []string{"216.238.121.95", "104.156.244.186", "45.32.218.87", "95.179.154.226"},
		RegistryPath:      "SOFTWARE\\Tibia Relic\\Credentials",
	},
}
