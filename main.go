package main

import (
	"ginchat/router"
	"ginchat/utils"
)

func init() {
	utils.InitConfig()
	utils.InitMySQL()
}
func main() {
	r := router.Router()
	r.Run("127.0.0.1:8080")
}
