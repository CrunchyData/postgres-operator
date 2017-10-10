package pvcservice

import (
	"github.com/emicklei/go-restful"
	"net/http"
)

type Pvc struct {
	Id, Name string
}

func New() *restful.WebService {
	service := new(restful.WebService)
	service.
		Path("/users").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_XML, restful.MIME_JSON)

	service.Route(service.GET("/{user-id}").To(FindPvc))
	service.Route(service.POST("").To(UpdatePvc))
	service.Route(service.PUT("/{user-id}").To(CreatePvc))
	service.Route(service.DELETE("/{user-id}").To(RemovePvc))

	return service
}
func FindPvc(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("user-id")
	// here you would fetch user from some persistence system
	usr := &Pvc{Id: id, Name: "John Doe"}
	response.WriteEntity(usr)
}
func UpdatePvc(request *restful.Request, response *restful.Response) {
	usr := new(Pvc)
	err := request.ReadEntity(&usr)
	// here you would update the user with some persistence system
	if err == nil {
		response.WriteEntity(usr)
	} else {
		response.WriteError(http.StatusInternalServerError, err)
	}
}

func CreatePvc(request *restful.Request, response *restful.Response) {
	usr := Pvc{Id: request.PathParameter("user-id")}
	err := request.ReadEntity(&usr)
	// here you would create the user with some persistence system
	if err == nil {
		response.WriteEntity(usr)
	} else {
		response.WriteError(http.StatusInternalServerError, err)
	}
}

func RemovePvc(request *restful.Request, response *restful.Response) {
	// here you would delete the user from some persistence system
}
