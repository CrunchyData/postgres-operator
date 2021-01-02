package cmd

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// dfTextPadding contains the values for what the text padding should be
type dfTextPadding struct {
	Capacity    int
	Instance    int
	Pod         int
	PercentUsed int
	PVC         int
	PVCType     int
	Used        int
}

// the different capacity levels and when to warn. Perhaps at some point one
// can configure this
const (
	capacityWarning = 85
	capacityCaution = 70
)

// values for the text paddings that remain constant
const (
	dfTextPaddingCapacity    = 10
	dfTextPaddingPVCType     = 12
	dfTextPaddingUsed        = 10
	dfTextPaddingPercentUsed = 7
)

// pvcTypeToString contains the human readable strings of the PVC types
var pvcTypeToString = map[msgs.DfPVCType]string{
	msgs.PVCTypePostgreSQL:    "data",
	msgs.PVCTypepgBackRest:    "pgbackrest",
	msgs.PVCTypeTablespace:    "tablespace",
	msgs.PVCTypeWriteAheadLog: "wal",
}

var dfCmd = &cobra.Command{
	Use:   "df",
	Short: "Display disk space for clusters",
	Long: `Displays the disk status for PostgreSQL clusters. For example:

	pgo df mycluster
	pgo df --selector=env=research
	pgo df --all`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("df called")

		// if the namespace is not set, use the namespcae loaded from the
		// environmental variable
		if Namespace == "" {
			Namespace = PGONamespace
		}

		// if the AllFlag is set, set the Selector to "*"
		if AllFlag {
			Selector = msgs.DfShowAllSelector
		}

		if Selector == "" && len(args) == 0 {
			fmt.Println(`Error: You must specify at least one cluster, selector, or use the "--all" flag.`)
			os.Exit(1)
		}

		// if there are multiple cluster names set, iterate through them
		// otherwise, just make the single API call
		if len(args) > 0 {
			for _, clusterName := range args {
				// set the selector
				selector := fmt.Sprintf("name=%s", clusterName)

				showDf(Namespace, selector)
			}
			return
		}

		showDf(Namespace, Selector)
	},
}

func init() {
	RootCmd.AddCommand(dfCmd)

	dfCmd.Flags().BoolVar(&AllFlag, "all", false, "Get disk utilization for all managed clusters")
	dfCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", `The output format. Supported types are: "json"`)
	dfCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")

}

// getPVCType returns a "human readable" form of the PVC
func getPVCType(pvcType msgs.DfPVCType) string {
	return pvcTypeToString[pvcType]
}

// getUtilizationColor returns the appropriate color to use on the terminal
// based on the overall utilization
func getUtilizationColor(utilization float64) func(...interface{}) string {
	// go through the levels and return the appropriate color
	switch {
	case utilization >= capacityWarning:
		return RED
	case utilization >= capacityCaution:
		return YELLOW
	default:
		return GREEN
	}
}

// makeDfInterface returns an interface slice of the available values in the df
func makeDfInterface(values []msgs.DfDetail) []interface{} {
	// iterate through the list of values to make the interface
	dfInterface := make([]interface{}, len(values))

	for i, value := range values {
		dfInterface[i] = value
	}

	return dfInterface
}

// printDfText renders a text response
func printDfText(response msgs.DfResponse) {
	// if the request errored, return the message here and exit with an error
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// if no results returned, return an error
	if len(response.Results) == 0 {
		fmt.Println("Nothing found.")
		return
	}

	// go get the max length of a few of the values we need to make an interface
	// slice
	dfInterface := makeDfInterface(response.Results)

	padding := dfTextPadding{
		Capacity:    dfTextPaddingCapacity,
		Instance:    getMaxLength(dfInterface, headingInstance, "InstanceName"),
		PercentUsed: dfTextPaddingPercentUsed,
		Pod:         getMaxLength(dfInterface, headingPod, "PodName"),
		PVC:         getMaxLength(dfInterface, headingPVC, "PVCName"),
		PVCType:     dfTextPaddingPVCType,
		Used:        dfTextPaddingUsed,
	}

	printDfTextHeader(padding)

	// sort the results!
	results := response.Results
	sort.SliceStable(results, func(i int, j int) bool {
		return results[i].InstanceName < results[j].InstanceName
	})

	// iterate through the reuslts and print them out
	for _, result := range results {
		printDfTextRow(result, padding)
	}
}

// printDfTextHeader prints out the header
func printDfTextHeader(padding dfTextPadding) {
	// print the header
	fmt.Println("")
	fmt.Printf("%s", util.Rpad(headingPVC, " ", padding.PVC))
	fmt.Printf("%s", util.Rpad(headingInstance, " ", padding.Instance))
	fmt.Printf("%s", util.Rpad(headingPod, " ", padding.Pod))
	fmt.Printf("%s", util.Rpad(headingPVCType, " ", padding.PVCType))
	fmt.Printf("%s", util.Rpad(headingUsed, " ", padding.Used))
	fmt.Printf("%s", util.Rpad(headingCapacity, " ", padding.Capacity))
	fmt.Println(headingPercentUsed)

	// print the layer below the header...which prints out a bunch of "-" that's
	// 1 less than the padding value
	fmt.Println(
		strings.Repeat("-", padding.PVC-1),
		strings.Repeat("-", padding.Instance-1),
		strings.Repeat("-", padding.Pod-1),
		strings.Repeat("-", padding.PVCType-1),
		strings.Repeat("-", padding.Used-1),
		strings.Repeat("-", padding.Capacity-1),
		strings.Repeat("-", padding.PercentUsed-1),
	)
}

// printDfTextRow prints a row of the text data. It also performs calculations
// that that are used to pretty up the rendering
func printDfTextRow(result msgs.DfDetail, padding dfTextPadding) {
	// get how the utilization and capacity should be render with their units
	pvcUsedSize, pvcUsedUnit := getSizeAndUnit(result.PVCUsed)
	pvcCapacitySize, pvcCapacityUnit := getSizeAndUnit(result.PVCCapacity)

	// perform some upfront calculations
	// get the % disk utilization
	utilization := math.Round(float64(result.PVCUsed) / float64(result.PVCCapacity) * 100)

	// get the color to give guidance on how much disk is being utilized
	utilizationColor := getUtilizationColor(utilization)

	fmt.Printf("%s", util.Rpad(result.PVCName, " ", padding.PVC))
	fmt.Printf("%s", util.Rpad(result.InstanceName, " ", padding.Instance))
	fmt.Printf("%s", util.Rpad(result.PodName, " ", padding.Pod))
	fmt.Printf("%s", util.Rpad(getPVCType(result.PVCType), " ", padding.PVCType))

	fmt.Printf("%s",
		util.Rpad(fmt.Sprintf("%.f%s", pvcUsedSize, getUnitString(pvcUsedUnit)), " ", padding.Used))

	fmt.Printf("%s",
		util.Rpad(fmt.Sprintf("%.f%s", pvcCapacitySize, getUnitString(pvcCapacityUnit)), " ", padding.Capacity))

	fmt.Printf("%s\n", utilizationColor(fmt.Sprintf("%.f%%", utilization)))
}

// showDf is the legacy function that handles processing the "pgo df" command
func showDf(namespace, selector string) {
	request := msgs.DfRequest{
		Namespace: namespace,
		Selector:  selector,
	}

	// make the request
	response, err := api.ShowDf(httpclient, &SessionCredentials, request)

	// if there is an error, or the response code is not ok, print the error and
	// exit
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	// render the next bit based on the output type
	switch OutputFormat {
	case "json":
		printJSON(response)
	default:
		printDfText(response)
	}
}
