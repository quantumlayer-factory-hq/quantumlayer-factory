package commands

import (
	"go.temporal.io/sdk/client"
)

func NewTemporalClient(addr, namespace string) (client.Client, error) {
	return client.Dial(client.Options{
		HostPort:  addr,
		Namespace: namespace,
	})
}