package templates 

import (
	"github.com/andig/evcc-config/registry"
)

func init() {
	template := registry.Template{
		Class:  "vehicle",
		Type:   "ovms",
		Name:   "OVMS (Twizzy, Smart, ...)",
		Sample: `title: Smart ED # display name for UI
capacity: 17 # kWh
user: # user server
password: # password server
vehicleid: # vehicle id
server: dexters-web.de # used ovms server [dexters-web.de or api.openvehicles.com]`,
	}

	registry.Add(template)
}
