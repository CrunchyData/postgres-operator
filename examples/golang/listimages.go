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

	//for _, container := range containers {
	//fmt.Printf("%s %s\n", container.ID[:10], container.Image)
	//}

	myfilters := filters.NewArgs()
	myfilters.Add("label", "Vendor=Crunchy Data Solutions")
	//myfilters.Add("label", "label=PostgresVersion=9.5")
	//fmt.Printf("filters are %v\n", myfilters)
	//mymap := make(map[string]string)
	//mymap["Vendor"] = "Crunchy Data Solutions"
	//mymap["PostgresVersion"] = "9.6"
	//myfilters.MatchKVList("label", mymap)

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
				//fmt.Printf("%s \n", name)
				//fmt.Printf("PostgresFullVersion %s PostgresVersion %s \n", image.Labels["PostgresFullVersion"], image.Labels["PostgresVersion"])
				pgmap[name] = image.Labels["PostgresFullVersion"]
			}
		}
	}
	fmt.Println(time.Now())
	fmt.Printf("%v\n", pgmap)
}
