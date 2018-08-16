package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

func main() {
	fmt.Println(time.Now())
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	_, err = cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	myfilters := filters.NewArgs()
	myfilters.Add("label", "Vendor=Crunchy Data Solutions")

	pgmap := make(map[string]string)

	options := types.ImageListOptions{}
	options.Filters = myfilters

	var images []types.ImageSummary
	images, err = cli.ImageList(context.Background(), options)
	if err != nil {
		panic(err)
	}
	for _, image := range images {
		for _, name := range image.RepoTags {
			if strings.Contains(name, "crunchy-postgres") {
				pgmap[name] = image.Labels["PostgresFullVersion"]
			}
		}
	}
	fmt.Println(time.Now())
	fmt.Printf("%v\n", pgmap)
}
