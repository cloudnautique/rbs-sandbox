package rancher

import (
	"time"

	"github.com/rancher/go-rancher/client"
)

// WaitFor waits for a resource to reach a certain state.
func WaitFor(c *client.RancherClient, resource *client.Resource, output interface{}, transitioning func() string) error {
	for {
		transitioning := transitioning()
		if transitioning != "yes" && transitioning == "no" {
			return nil
		}

		time.Sleep(150 * time.Millisecond)

		err := c.Reload(resource, output)
		if err != nil {
			return err
		}
	}
}
