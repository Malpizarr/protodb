package api

import (
	"net/http"

	"github.com/Malpizarr/dbproto/data"
)

func SetupRoutes(server *data.Server) {
	http.HandleFunc("/createDatabase", CreateDatabaseHandler(server))
	http.HandleFunc("/createTable", CreateTableHandler(server))
	http.HandleFunc("/listDatabases", ListDatabasesHandler(server))
	http.HandleFunc("/tableAction", TableActionHandler(server))
}