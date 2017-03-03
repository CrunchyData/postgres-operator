package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	//_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type CrunchyDatabaseSpec struct {
	Foo string `json:"foo"`
	Bar bool   `json:"bar"`
}

type CrunchyDatabase struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec CrunchyDatabaseSpec `json:"spec"`
}

type CrunchyDatabaseList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []CrunchyDatabase `json:"items"`
}

func (e *CrunchyDatabase) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *CrunchyDatabase) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (el *CrunchyDatabaseList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
}

func (el *CrunchyDatabaseList) GetListMeta() unversioned.List {
	return &el.Metadata
}

type CrunchyDatabaseListCopy CrunchyDatabaseList
type CrunchyDatabaseCopy CrunchyDatabase

func (e *CrunchyDatabase) UnmarshalJSON(data []byte) error {
	tmp := CrunchyDatabaseCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := CrunchyDatabase(tmp)
	*e = tmp2
	return nil
}

func (el *CrunchyDatabaseList) UnmarshalJSON(data []byte) error {
	tmp := CrunchyDatabaseListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := CrunchyDatabaseList(tmp)
	*el = tmp2
	return nil
}

var (
	config *rest.Config
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "the path to a kubeconfig, specifies this tool runs outside the cluster")
	flag.Parse()

	client, err := buildClientFromFlags(*kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	exampleList := CrunchyDatabaseList{}
	err = client.Get().
		Resource("crunchydatabases").
		Do().Into(&exampleList)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("%#v\n", exampleList)

	example := CrunchyDatabase{}
	err = client.Get().
		Namespace("default").
		Resource("crunchydatabases").
		Name("example1").
		Do().Into(&example)
	if err != nil {
		fmt.Println("example1 not found")
	} else {
		fmt.Printf("%#v\n", example)
	}

	fmt.Println()
	fmt.Println("---------------------------------------------------------")
	fmt.Println()

	eventchan := make(chan *CrunchyDatabase)
	stopchan := make(chan struct{}, 1)
	source := cache.NewListWatchFromClient(client, "crunchydatabases", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		example := obj.(*CrunchyDatabase)
		eventchan <- example
		fmt.Println("creating an example object")
		fmt.Println("created with Foo=" + example.Spec.Foo)
	}
	createDeleteHandler := func(obj interface{}) {
		example := obj.(*CrunchyDatabase)
		eventchan <- example
		fmt.Println("deleting an example object")
		fmt.Println("deleted with Foo=" + example.Spec.Foo)
	}

	updateHandler := func(old interface{}, obj interface{}) {
		example := obj.(*CrunchyDatabase)
		eventchan <- example
		fmt.Println("updating an example object")
		fmt.Println("updated with Foo=" + example.Spec.Foo)
	}

	_, controller := cache.NewInformer(
		source,
		&CrunchyDatabase{},
		time.Second*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    createAddHandler,
			UpdateFunc: updateHandler,
			DeleteFunc: createDeleteHandler,
		})

	go controller.Run(stopchan)

	go func() {
		for {
			select {
			case event := <-eventchan:
				fmt.Printf("%#v\n", event)
			}
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case s := <-signals:
			fmt.Printf("received signal %#v, exiting...\n", s)
			os.Exit(0)
		}
	}
}

func buildClientFromFlags(kubeconfig string) (*rest.RESTClient, error) {
	config, err := configFromFlags(kubeconfig)
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &unversioned.GroupVersion{
		Group:   "crunchydata.com",
		Version: "v1",
	}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	schemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	schemeBuilder.AddToScheme(api.Scheme)

	return rest.RESTClientFor(config)
}

func configFromFlags(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		unversioned.GroupVersion{Group: "crunchydata.com", Version: "v1"},
		&CrunchyDatabase{},
		&CrunchyDatabaseList{},
		&api.ListOptions{},
		&api.DeleteOptions{},
	)

	return nil
}
